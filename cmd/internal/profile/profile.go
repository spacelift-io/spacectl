package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/client/session"
)

var (
	configDir   string
	currentPath string
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

			configDir = filepath.Join(homeDir, session.SpaceliftConfigDirectory)
			currentPath = filepath.Join(configDir, session.CurrentFileName)

			if err := os.MkdirAll(configDir, 0700); err != nil {
				return fmt.Errorf("could not create Spacelift config directory: %w", err)
			}

			return nil
		},
		Subcommands: []*cli.Command{
			loginCommand(),
			logoutCommand(),
			selectCommand(),
		},
	}
}
