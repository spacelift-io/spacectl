package authenticated

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/term"

	"github.com/spacelift-io/spacectl/browserauth"
	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
)

func LoginCurrentProfile(ctx context.Context) error {
	manager, err := session.UserProfileManager()
	if err != nil {
		return fmt.Errorf("could not accesss profile manager: %w", err)
	}

	profile := manager.Current()
	if profile == nil {
		return fmt.Errorf("no current profile set, please use `spacectl profile select <alias>` to select a profile")
	}

	fmt.Println("===========================================================")
	fmt.Println("Logging in to the current profile:", profile.Alias)
	fmt.Println("Detected profile type:", profile.Credentials.Type)
	fmt.Println("===========================================================")

	storedCredentials := profile.Credentials
	cfg := &BrowserConfig{}
	if err := LoginByType(ctx, storedCredentials, cfg); err != nil {
		return fmt.Errorf("could not login: %w", err)
	}

	// Check if the credentials are valid before we try persisting them.
	if _, err := storedCredentials.Session(ctx, client.GetHTTPClient()); err != nil {
		return fmt.Errorf("credentials look invalid: %w", err)
	}

	return manager.Create(&session.Profile{
		Alias:       profile.Alias,
		Credentials: storedCredentials,
	})
}

func LoginByType(ctx context.Context, creds *session.StoredCredentials, browserCfg *BrowserConfig) error {
	switch creds.Type {
	case session.CredentialsTypeAPIKey:
		reader := bufio.NewReader(os.Stdin)
		if err := LoginUsingAPIKey(reader, creds); err != nil {
			return err
		}
	case session.CredentialsTypeGitHubToken:
		if err := LoginUsingGitHubAccessToken(creds); err != nil {
			return err
		}
	case session.CredentialsTypeAPIToken:
		return LoginUsingWebBrowser(ctx, creds, browserCfg)
	default:
		return fmt.Errorf("invalid selection (%s), please try again", creds.Type)
	}

	return nil
}

func LoginUsingAPIKey(reader *bufio.Reader, creds *session.StoredCredentials) error {
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

func LoginUsingGitHubAccessToken(creds *session.StoredCredentials) error {
	fmt.Print("Enter GitHub access token: ")

	accessToken, err := term.ReadPassword(int(syscall.Stdin)) //nolint: unconvert
	if err != nil {
		return err
	}
	creds.AccessToken = strings.TrimSpace(string(accessToken))

	fmt.Println()

	return nil
}

func LoginUsingWebBrowser(ctx context.Context, creds *session.StoredCredentials, cfg *BrowserConfig) error {
	handler, err := browserauth.BeginWithBindAddress(creds, cfg.getBindHost(), cfg.getBindPort())
	if err != nil {
		return err
	}

	fmt.Printf("Waiting for login responses at %s:%d\n", handler.Host, handler.Port)
	fmt.Printf("\nOpening browser to %s\n\n", handler.AuthenticationURL)

	if err := browser.OpenURL(handler.AuthenticationURL); err != nil {
		fmt.Printf("Failed to open the browser: %s\nPlease open the URL manually\n\n", err.Error())
	}

	fmt.Println("Waiting for login...")

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := handler.Wait(waitCtx); err != nil {
		return err
	}

	fmt.Println("Done!")

	return nil
}

type BrowserConfig struct {
	BindHost *string
	BindPort *int
}

func (b *BrowserConfig) getBindHost() string {
	if b.BindHost == nil {
		return "localhost"
	}
	return *b.BindHost
}

func (b *BrowserConfig) getBindPort() int {
	if b.BindPort == nil {
		return 0
	}
	return *b.BindPort
}
