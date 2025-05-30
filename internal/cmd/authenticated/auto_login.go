package authenticated

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func AttemptAutoLogin(ctx context.Context, _ *cli.Command) (context.Context, error) {
	autoLogin(ctx)
	return ctx, nil
}

func autoLogin(ctx context.Context) {
	if os.Getenv("SPACELIFT_AUTO_LOGIN") == "" {
		return
	}

	ctx, err := Ensure(ctx, nil)
	if err != nil {
		fmt.Printf("Error: %s\nFailed to check if logged in, will assume true\n\n", err)
		return
	}

	_, err = CurrentViewer(ctx)
	if err == nil {
		return
	}
	if errors.Is(err, ErrViewerUnknown) {
		fmt.Println("Attempting auto-login...")
		if err := LoginCurrentProfile(ctx); err != nil {
			fmt.Printf("Error: %s\nAuto-login failed, attempt to login manually\n\n", err)
		}
		return
	}

	fmt.Printf("Error: %s\nFailed to check if logged in, will assume true\n\n", err)
}
