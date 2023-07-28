package profile

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

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
			flagBindHost,
			flagBindPort,
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

	fmt.Print("Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/): ")

	endpoint, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read Spacelift endpoint: %w", err)
	}
	endpoint = strings.TrimSpace(endpoint)

	if endpoint == "" {
		return errors.New("Spacelift endpoint cannot be empty")
	}

	url, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return fmt.Errorf("invalid Spacelift endpoint: %w", err)
	}
	if url.Scheme == "" || url.Host == "" {
		return fmt.Errorf("scheme and host must be valid: parsed scheme %q and host %q", url.Scheme, url.Host)
	}

	storedCredentials.Endpoint = endpoint

Loop:
	for {
		fmt.Printf(
			"Select authentication flow: \n  %d) for API key,\n  %d) for GitHub access token,\n  %d) for login with a web browser\nOption: ",
			session.CredentialsTypeAPIKey,
			session.CredentialsTypeGitHubToken,
			session.CredentialsTypeAPIToken,
		)

		credentialsType, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("could not read Spacelift credentials type: %w", err)
		}

		typeNum, err := strconv.Atoi(strings.TrimSpace(credentialsType))
		if err != nil {
			fmt.Printf("Invalid selection (%s), please try again", credentialsType)
			continue
		}

		storedCredentials.Type = session.CredentialsType(typeNum)

		switch storedCredentials.Type {
		case session.CredentialsTypeAPIKey:
			if err := loginUsingAPIKey(reader, &storedCredentials); err != nil {
				return err
			}
			break Loop
		case session.CredentialsTypeGitHubToken:
			if err := loginUsingGitHubAccessToken(&storedCredentials); err != nil {
				return err
			}
			break Loop
		case session.CredentialsTypeAPIToken:
			return loginUsingWebBrowser(ctx, &storedCredentials)
		default:
			fmt.Printf("Invalid selection (%s), please try again", credentialsType)
			continue
		}
	}

	// Check if the credentials are valid before we try persisting them.
	if _, err := storedCredentials.Session(session.Defaults()); err != nil {
		return fmt.Errorf("credentials look invalid: %w", err)
	}

	return persistAccessCredentials(&storedCredentials)
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
			log.Fatal(err)
		}

		if handlerErr != nil {
			log.Println(handlerErr)
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
				log.Fatalf("could not start local server: %s", err)
			}
		}
	}()

	return server, bound, nil
}
