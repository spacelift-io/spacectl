package account

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Log out of an existing Spacelift account",
		Action: func(*cli.Context) error {
			if _, err := os.Stat(aliasPath); err != nil {
				return fmt.Errorf("you don't seem to be logged in to %s: %v", accountAlias, err)
			}

			if err := os.Remove(aliasPath); err != nil {
				return err
			}

			currentTarget, err := os.Readlink(currentPath)

			switch {
			case os.IsNotExist(err):
				return nil
			case err == nil && currentTarget == aliasPath:
				return os.Remove(currentPath)
			default:
				return err
			}
		},
	}
}
