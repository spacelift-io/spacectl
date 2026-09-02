package authenticated

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"

	"github.com/spacelift-io/spacectl/browserauth"
	"github.com/spacelift-io/spacectl/client/session"
)

const (
	envNoAutoLogin = "SPACECTL_NO_AUTO_LOGIN"
	envBindHost    = "SPACECTL_BIND_HOST"
	envBindPort    = "SPACECTL_BIND_PORT"
)

func maybeReloginExpiredBrowserSession(ctx context.Context) error {
	if !shouldAttemptBrowserRelogin(isInteractive()) {
		return nil
	}

	manager, err := session.UserProfileManager()
	if err != nil {
		return fmt.Errorf("could not access profile manager: %w", err)
	}

	profile := manager.Current()
	if profile == nil || !profile.Credentials.APITokenExpired() {
		return nil
	}

	if err := confirmRelogin(); err != nil {
		return err
	}

	host, port := bindAddress()
	if err := browserauth.Login(ctx, profile.Credentials, host, port, false); err != nil {
		return fmt.Errorf("browser login failed: %w", err)
	}

	if err := manager.Create(profile); err != nil {
		return fmt.Errorf("could not persist profile: %w", err)
	}

	return nil
}

func shouldAttemptBrowserRelogin(interactive bool) bool {
	if !interactive || session.EnvironmentConfigured() {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv(envNoAutoLogin))) {
	case "1", "true", "yes":
		return false
	default:
		return true
	}
}

func confirmRelogin() error {
	prompt := promptui.Prompt{
		Label:     "Session expired. Log in via browser",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrAbort) || errors.Is(err, promptui.ErrInterrupt) {
			return errors.New("login cancelled; run `spacectl profile login` to authenticate")
		}
		return fmt.Errorf("could not confirm login: %w", err)
	}

	return nil
}

func isInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

func bindAddress() (string, int) {
	host := os.Getenv(envBindHost)
	if host == "" {
		host = "localhost"
	}

	port := 0
	if raw := os.Getenv(envBindPort); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			port = n
		}
	}

	return host, port
}
