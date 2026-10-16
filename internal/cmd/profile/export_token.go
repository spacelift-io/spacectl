package profile

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func exportTokenCommand() *cli.Command {
	return &cli.Command{
		Name: "export-token",
		Usage: "Prints the current token to stdout. In order not to leak, " +
			"we suggest piping it to your OS pastebin",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(ctx context.Context, _ *cli.Command) error {
			httpClient, err := authenticated.HTTPClient()
			if err != nil {
				return err
			}

			sess, err := session.FromProfileOrEnvironment(ctx, manager, httpClient, os.LookupEnv)
			if err != nil {
				return fmt.Errorf("could not get session: %w", err)
			}

			token, err := sess.BearerToken(ctx)
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
