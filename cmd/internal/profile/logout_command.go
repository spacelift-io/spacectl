package profile

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:   "logout",
		Usage:  "Remove Spacelift credentials for an existing profile",
		Before: getAlias,
		Action: func(*cli.Context) error {
			if _, err := os.Stat(aliasPath); err != nil {
				return fmt.Errorf("you don't seem to be have any Spacelift credentials associated with %s: %v", profileAlias, err)
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
