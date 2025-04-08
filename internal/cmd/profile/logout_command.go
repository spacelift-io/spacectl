package profile

import (
	"context"

	"github.com/urfave/cli/v3"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Remove Spacelift credentials for an existing profile",
		ArgsUsage: "<account-alias>",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			_, err := setGlobalProfileAlias(cmd)
			return ctx, err
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return manager.Delete(profileAlias)
		},
	}
}
