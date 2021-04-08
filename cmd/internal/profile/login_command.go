package profile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/spacelift-io/spacelift-cli/client/session"
)

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:   "login",
		Usage:  "Create a profile for a Spacelift account",
		Before: getAlias,
		Action: loginAction,
	}
}

func loginAction(*cli.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/): ")

	var storedCredentials session.StoredCredentials

	endpoint, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("could not read Spacelift endpoint: %w", err)
	}
	storedCredentials.Endpoint = strings.TrimSpace(endpoint)

Loop:
	for {
		fmt.Printf(
			"Select credentials type: %d for API key, %d for GitHub access token: ",
			session.CredentialsTypeAPIKey,
			session.CredentialsTypeGitHubToken,
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

func persistAccessCredentials(creds *session.StoredCredentials) error {
	file, err := os.OpenFile(aliasPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not create config file for %s: %w", profileAlias, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(creds); err != nil {
		return fmt.Errorf("could not write config file for %s: %w", profileAlias, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("could close the config file for %s: %w", profileAlias, err)
	}

	return setCurrentProfile()
}
