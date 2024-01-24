package profile

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
)

var (
	manager *session.ProfileManager
)

// Command encapsulates the profile command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage Spacelift profiles",
		Before: func(cliCtx *cli.Context) error {
			var err error
			if manager, err = session.UserProfileManager(); err != nil {
				return fmt.Errorf("could not initialize profile manager: %w", err)
			}
			return nil
		},
		Subcommands: []*cli.Command{
			currentCommand(),
			exportTokenCommand(),
			usageViewCSVCommand(),
			listCommand(),
			loginCommand(),
			logoutCommand(),
			selectCommand(),
		},
	}
}
