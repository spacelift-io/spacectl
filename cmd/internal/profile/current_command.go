package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func currentCommand() *cli.Command {
	return &cli.Command{
		Name:  "current",
		Usage: "Outputs the account you currently have selected",

		// Use a space to cause the args usage to not be displayed since the `current` command
		// doesn't accept any arguments
		ArgsUsage: " ",
		Action: func(ctx *cli.Context) error {
			if _, err := os.Lstat(currentPath); err != nil {
				return fmt.Errorf("no account is currently selected: %w", err)
			}

			linkTarget, err := os.Readlink(currentPath)
			if err != nil {
				return fmt.Errorf("could not find the target of the current account symlink: %w", err)
			}

			alias := filepath.Base(linkTarget)
			fmt.Println(alias)

			return nil
		},
	}
}
