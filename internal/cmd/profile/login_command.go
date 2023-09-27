package profile

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal"
)

const (
	cliBrowserPath     = "/cli_login"
	cliAuthSuccessPage = "/auth_success"
	cliAuthFailurePage = "/auth_failure"
)

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:      "login",
		Usage:     "Create a profile for a Spacelift account",
		Before:    getAliasWithAPITokenProfile,
		ArgsUsage: "<account-alias>",
		Action:    loginAction,
		Flags: []cli.Flag{
			flagMethod,
			flagBindHost,
			flagBindPort,
			flagEndpoint,
		},
	}
}

func loginAction(ctx *cli.Context) error {
	var storedCredentials session.StoredCredentials

	// Let's try to re-authenticate user.
	if apiTokenProfile != nil {
		storedCredentials.Endpoint = apiTokenProfile.Credentials.Endpoint
		storedCredentials.Type = apiTokenProfile.Credentials.Type
		profileAlias = apiTokenProfile.Alias

		return loginUsingWebBrowser(ctx, &storedCredentials)
	}

	reader := bufio.NewReader(os.Stdin)

	endpoint, err := readEndpoint(ctx, reader)
	if err != nil {
		return err
	}
	storedCredentials.Endpoint = endpoint

	credentialsType, err := getCredentialsType(ctx)
	if err != nil {
		return err
	}
	storedCredentials.Type = credentialsType

	switch storedCredentials.Type {
	case session.CredentialsTypeAPIKey:
		if err := loginUsingAPIKey(reader, &storedCredentials); err != nil {
			return err
		}
	case session.CredentialsTypeGitHubToken:
		if err := loginUsingGitHubAccessToken(&storedCredentials); err != nil {
			return err
		}
	case session.CredentialsTypeAPIToken:
		return loginUsingWebBrowser(ctx, &storedCredentials)
	default:
		return fmt.Errorf("invalid selection (%s), please try again", storedCredentials.Type)
	}

	// Check if the credentials are valid before we try persisting them.
	if _, err := storedCredentials.Session(session.Defaults()); err != nil {
		return fmt.Errorf("credentials look invalid: %w", err)
	}

	return persistAccessCredentials(&storedCredentials)
}

func getCredentialsType(ctx *cli.Context) (session.CredentialsType, error) {
	if ctx.IsSet(flagMethod.Name) {
		got := methodToCredentialsType[ctx.String(flagMethod.Name)]
		return session.CredentialsType(got), nil
	}

	prompt := promptui.Select{
		Label: "Select authentication flow:",
		Items: []string{"API key", "GitHub access token", "Login with a web browser"},
		Size:  3,
	}
	result, _, err := prompt.Run()
	if err != nil {
		return 0, err
	}

	return session.CredentialsType(result + 1), nil
}

func readEndpoint(ctx *cli.Context, reader *bufio.Reader) (string, error) {
	var endpoint string
	if ctx.IsSet(flagEndpoint.Name) {
		endpoint = ctx.String(flagEndpoint.Name)
	} else {
		fmt.Print("Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/): ")

		var err error
		endpoint, err = reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("could not read Spacelift endpoint: %w", err)
		}

		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			return "", errors.New("Spacelift endpoint cannot be empty")
		}
	}

	url, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid Spacelift endpoint: %w", err)
	}
	if url.Scheme == "" || url.Host == "" {
		return "", fmt.Errorf("scheme and host must be valid: parsed scheme %q and host %q", url.Scheme, url.Host)
	}

	return endpoint, nil
}

func loginUsingAPIKey(reader *bufio.Reader, creds *session.StoredCredentials) error {
	fmt.Print("Enter API key ID: ")
	keyID, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	creds.KeyID = strings.TrimSpace(keyID)

	fmt.Print("Enter API key secret: ")
	keySecret, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	creds.KeySecret = strings.TrimSpace(string(keySecret))

	fmt.Println()

	return nil
}

func loginUsingGitHubAccessToken(creds *session.StoredCredentials) error {
	fmt.Print("Enter GitHub access token: ")

	accessToken, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	creds.AccessToken = strings.TrimSpace(string(accessToken))

	fmt.Println()

	return nil
}

func loginUsingWebBrowser(ctx *cli.Context, creds *session.StoredCredentials) error {
	pubKey, privKey, err := internal.GenerateRSAKeyPair()
	if err != nil {
		return errors.Wrap(err, "could not generate RSA key pair")
	}

	keyBase64 := base64.RawURLEncoding.EncodeToString(pubKey)

	done := make(chan bool, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerErr := func() error {
			base64Token := r.URL.Query().Get("token")
			if base64Token == "" {
				return errors.New("missing token parameter")
			}

			base64Key := r.URL.Query().Get("key")
			if base64Key == "" {
				return errors.New("missing key parameter")
			}

			encToken, err := base64.RawURLEncoding.DecodeString(base64Token)
			if err != nil {
				return errors.Wrap(err, "could not decode session token")
			}

			encKey, err := base64.RawURLEncoding.DecodeString(base64Key)
			if err != nil {
				return errors.Wrap(err, "could not decode key")
			}

			key, err := internal.DecryptRSA(privKey, []byte(encKey))
			if err != nil {
				return errors.Wrap(err, "could not decrypt key")
			}

			jwt, err := internal.DecryptAES(key, []byte(encToken))
			if err != nil {
				return errors.Wrap(err, "could not decrypt session token")
			}

			creds.AccessToken = string(jwt)

			return persistAccessCredentials(creds)
		}()

		infoPage, err := url.Parse(creds.Endpoint)
		if err != nil {
			slog.Error("Error parsing URL", "err", err)
			os.Exit(1)
		}

		if handlerErr != nil {
			slog.Error("login error", "err", handlerErr)
			infoPage.Path = cliAuthFailurePage
			http.Redirect(w, r, infoPage.String(), http.StatusTemporaryRedirect)
		} else {
			fmt.Println("Done!")
			infoPage.Path = cliAuthSuccessPage
			http.Redirect(w, r, infoPage.String(), http.StatusTemporaryRedirect)
		}

		done <- true
	}

	server, port, err := serveOnOpenPort(&bindHost, &bindPort, handler)
	if err != nil {
		return err
	}

	browserURL, err := buildBrowserURL(creds.Endpoint, keyBase64, port)
	if err != nil {
		server.Close()
		return errors.Wrap(err, "could not build browser URL")
	}

	fmt.Printf("\nOpening browser to %s\n\n", browserURL)

	if err := browser.OpenURL(browserURL); err != nil {
		fmt.Printf("Failed to open the browser: %s\nPlease open the URL manually\n\n", err.Error())
	}

	fmt.Println("Waiting for login...")

	select {
	case <-done:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return errors.Wrap(server.Shutdown(shutdownCtx), "could not stop the server")
	case <-time.After(2 * time.Minute):
		server.Close()
		return errors.New("login timeout exceeded")
	}
}

func buildBrowserURL(endpoint, pubKey string, port int) (string, error) {
	base, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	base.Path = cliBrowserPath

	q := url.Values{}
	q.Add("key", pubKey)
	q.Add("port", fmt.Sprint(port))

	base.RawQuery = q.Encode()

	return base.String(), nil
}

func persistAccessCredentials(creds *session.StoredCredentials) error {
	return manager.Create(&session.Profile{
		Alias:       profileAlias,
		Credentials: creds,
	})
}

func serveOnOpenPort(host *string, port *int, handler func(w http.ResponseWriter, r *http.Request)) (*http.Server, int, error) {

	bindOn := fmt.Sprintf("%s:%d", *host, *port)

	addr, err := net.ResolveTCPAddr("tcp", bindOn)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to resolve tcp address")
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to start listening on %s", addr.String())
	}

	m := http.NewServeMux()
	m.HandleFunc("/", handler)

	bound := l.Addr().(*net.TCPAddr).Port
	fmt.Printf("Waiting for login responses at %v\n", l.Addr())

	server := &http.Server{Handler: m, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(l); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("could not start local server", "err", err)
				os.Exit(1)
			}
		}
	}()

	return server, bound, nil
}
