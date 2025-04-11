package profile

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/spacelift-io/spacectl/browserauth"
	"github.com/spacelift-io/spacectl/client/session"
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
		return got, nil
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

	return session.CredentialsType(result + 1), nil //nolint: gosec
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
	keySecret, err := term.ReadPassword(int(syscall.Stdin)) //nolint: unconvert
	if err != nil {
		return err
	}
	creds.KeySecret = strings.TrimSpace(string(keySecret))

	fmt.Println()

	return nil
}

func loginUsingGitHubAccessToken(creds *session.StoredCredentials) error {
	fmt.Print("Enter GitHub access token: ")

	accessToken, err := term.ReadPassword(int(syscall.Stdin)) //nolint: unconvert
	if err != nil {
		return err
	}
	creds.AccessToken = strings.TrimSpace(string(accessToken))

	fmt.Println()

	return nil
}

func loginUsingWebBrowser(_ *cli.Context, creds *session.StoredCredentials) error {
	// Begin the interactive browser auth flow
	handler, err := browserauth.BeginWithBindAddress(creds, bindHost, bindPort)
	if err != nil {
		return err
	}

	fmt.Printf("Waiting for login responses at %s:%d\n", handler.Host, handler.Port)
	fmt.Printf("\nOpening browser to %s\n\n", handler.AuthenticationURL)

	// Attempt to automatically open the URL in the user's browser
	if err := browser.OpenURL(handler.AuthenticationURL); err != nil {
		fmt.Printf("Failed to open the browser: %s\nPlease open the URL manually\n\n", err.Error())
	}

	fmt.Println("Waiting for login...")

	// Create a context that will timeout after 2 minutes while we wait for auth completion
	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Wait for the timeout or an auth callback
	if err := handler.Wait(waitCtx); err != nil {
		return err
	}

	// Save the shiny new token
	if err := persistAccessCredentials(creds); err != nil {
		return err
	}

	fmt.Println("Done!")

	return nil
}

func persistAccessCredentials(creds *session.StoredCredentials) error {
	return manager.Create(&session.Profile{
		Alias:       profileAlias,
		Credentials: creds,
	})
}
