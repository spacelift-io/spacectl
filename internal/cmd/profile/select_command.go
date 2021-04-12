package profile

import (
	"github.com/urfave/cli/v2"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:      "select",
		Usage:     "Select one of your Spacelift account profiles",
		ArgsUsage: "<account-alias>",
		Before:    getAlias,
		Action: func(*cli.Context) error {
			return manager.Select(profileAlias)
		},
	}
}
