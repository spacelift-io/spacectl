package profile

import (
	"github.com/urfave/cli/v2"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Remove Spacelift credentials for an existing profile",
		ArgsUsage: "<account-alias>",
		Before:    getAlias,
		Action: func(*cli.Context) error {
			return manager.Delete(profileAlias)
		},
	}
}
