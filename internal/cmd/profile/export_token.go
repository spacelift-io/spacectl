package profile

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/urfave/cli/v2"
)

func exportTokenCommand() *cli.Command {
	return &cli.Command{
		Name: "export-token",
		Usage: "Prints the current token to stdout. In order not to leak, " +
			"we suggest piping it to your OS pastebin",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(ctx *cli.Context) error {
			currentProfile := manager.Current()
			if currentProfile == nil {
				return errors.New("no account is currently selected")
			}

			session, err := currentProfile.Credentials.Session(ctx.Context, http.DefaultClient)
			if err != nil {
				return fmt.Errorf("could not get session: %w", err)
			}

			token, err := session.BearerToken(ctx.Context)
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
