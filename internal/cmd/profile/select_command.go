package profile

import (
	"context"

	"github.com/urfave/cli/v3"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:      "select",
		Usage:     "Select one of your Spacelift account profiles",
		ArgsUsage: "<account-alias>",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			_, err := setGlobalProfileAlias(cmd)
			return ctx, err
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return manager.Select(profileAlias)
		},
	}
}
