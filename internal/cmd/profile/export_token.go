package profile

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
)

func exportTokenCommand() *cli.Command {
	return &cli.Command{
		Name: "export-token",
		Usage: "Prints the current token to stdout. In order not to leak, " +
			"we suggest piping it to your OS pastebin",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			currentProfile := manager.Current()
			if currentProfile == nil {
				return errors.New("no account is currently selected")
			}

			session, err := currentProfile.Credentials.Session(ctx, http.DefaultClient)
			if err != nil {
				return fmt.Errorf("could not get session: %w", err)
			}

			token, err := session.BearerToken(ctx)
			if err != nil {
				return fmt.Errorf("could not get bearer token: %w", err)
			}

			if _, err = fmt.Print(token); err != nil {
				return fmt.Errorf("could not print token: %w", err)
			}

			return nil
		},
	}
}
