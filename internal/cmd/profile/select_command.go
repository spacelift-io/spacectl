package profile

import (
	"github.com/urfave/cli/v2"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:      "select",
		Usage:     "Select one of your Spacelift account profiles",
		ArgsUsage: "<account-alias>",
		Before: func(cliCtx *cli.Context) error {
			_, err := setGlobalProfileAlias(cliCtx)
			return err
		},
		Action: func(*cli.Context) error {
			return manager.Select(profileAlias)
		},
	}
}
