package browserauth

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/browser"

	"github.com/spacelift-io/spacectl/client/session"
)

const loginTimeout = 2 * time.Minute

func Login(ctx context.Context, creds *session.StoredCredentials, host string, port int, noBrowser bool) error {
	if host == "" {
		host = "localhost"
	}

	handler, err := BeginWithBindAddress(ctx, creds, host, port)
	if err != nil {
		return err
	}

	fmt.Printf("Waiting for login responses at %s:%d\n", handler.Host, handler.Port)
	fmt.Printf("\nOpening browser to %s\n\n", handler.AuthenticationURL)

	if noBrowser {
		fmt.Printf("Please open the following URL in your browser to complete login:\n%s\n\n", handler.AuthenticationURL)
	} else if err := browser.OpenURL(handler.AuthenticationURL); err != nil {
		fmt.Printf("Failed to open the browser: %s\nPlease open the URL manually\n\n", err.Error())
	}

	fmt.Println("Waiting for login...")

	waitCtx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	return handler.Wait(waitCtx)
}
