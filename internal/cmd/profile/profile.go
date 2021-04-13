package profile

import (
	"fmt"
	"os"
	"path/filepath"

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
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not get user home directory: %w", err)
			}

			configDir := filepath.Join(homeDir, session.SpaceliftConfigDirectory)
			if manager, err = session.NewProfileManager(configDir); err != nil {
				return fmt.Errorf("could not initialise profile manager: %w", err)
			}

			return nil
		},
		Subcommands: []*cli.Command{
			currentCommand(),
			listCommand(),
			loginCommand(),
			logoutCommand(),
			selectCommand(),
		},
	}
}
