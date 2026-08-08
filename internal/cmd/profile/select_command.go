package profile

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:      "select",
		Usage:     "Select one of your Spacelift account profiles",
		ArgsUsage: "<account-alias>",
		Before: func(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
			_, err := setGlobalProfileAlias(cliCmd)
			return ctx, err
		},
		Action: func(ctx context.Context, cliCmd *cli.Command) error {
			if err := manager.Select(profileAlias); err != nil {
				return err
			}

			// The selection was persisted, but this shell won't act on it, so say so rather
			// than letting the user believe the switch took effect.
			if override, ok := manager.ProfileOverride(); ok && override != profileAlias {
				fmt.Printf(
					"WARNING: %s is set to '%s', which takes precedence over the profile you just selected.\n"+
						"Unset %s for '%s' to take effect in this shell.\n",
					session.EnvSpaceliftProfile, override, session.EnvSpaceliftProfile, profileAlias,
				)
			}

			return nil
		},
	}
}
