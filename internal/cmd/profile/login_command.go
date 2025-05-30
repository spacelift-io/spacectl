package profile

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
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

func loginAction(ctx context.Context, cliCmd *cli.Command) error {
	var storedCredentials session.StoredCredentials

	browserCfg := &authenticated.BrowserConfig{
		BindHost: &bindHost,
		BindPort: &bindPort,
	}
	// Let's try to re-authenticate user.
	if apiTokenProfile != nil {
		storedCredentials.Endpoint = apiTokenProfile.Credentials.Endpoint
		storedCredentials.Type = apiTokenProfile.Credentials.Type
		profileAlias = apiTokenProfile.Alias

		if err := authenticated.LoginUsingWebBrowser(ctx, &storedCredentials, browserCfg); err != nil {
			return err
		}

		return persistAccessCredentials(&storedCredentials)
	}

	reader := bufio.NewReader(os.Stdin)

	endpoint, err := readEndpoint(cliCmd, reader)
	if err != nil {
		return err
	}
	storedCredentials.Endpoint = endpoint

	credentialsType, err := getCredentialsType(cliCmd)
	if err != nil {
		return err
	}
	storedCredentials.Type = credentialsType

	if err := authenticated.LoginByType(ctx, &storedCredentials, browserCfg); err != nil {
		return fmt.Errorf("could not login: %w", err)
	}

	// Check if the credentials are valid before we try persisting them.
	if _, err := storedCredentials.Session(ctx, client.GetHTTPClient()); err != nil {
		return fmt.Errorf("credentials look invalid: %w", err)
	}

	fmt.Println("Consider setting SPACELIFT_AUTO_LOGIN=true for automatic login in the future")
	return persistAccessCredentials(&storedCredentials)
}

func getCredentialsType(cliCmd *cli.Command) (session.CredentialsType, error) {
	if cliCmd.IsSet(flagMethod.Name) {
		got := methodToCredentialsType[cliCmd.String(flagMethod.Name)]
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

func readEndpoint(cliCmd *cli.Command, reader *bufio.Reader) (string, error) {
	var endpoint string
	if cliCmd.IsSet(flagEndpoint.Name) {
		endpoint = cliCmd.String(flagEndpoint.Name)
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

func persistAccessCredentials(creds *session.StoredCredentials) error {
	return manager.Create(&session.Profile{
		Alias:       profileAlias,
		Credentials: creds,
	})
}
