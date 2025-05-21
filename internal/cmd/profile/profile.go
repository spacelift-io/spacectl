package profile

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
)

var (
	manager *session.ProfileManager
)

// Command encapsulates the profile command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage Spacelift profiles",
		Before: func(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
			var err error
			if manager, err = session.UserProfileManager(); err != nil {
				return ctx, fmt.Errorf("could not initialize profile manager: %w", err)
			}
			return ctx, nil
		},
		Commands: []*cli.Command{
			currentCommand(),
			exportTokenCommand(),
			usageViewCSVCommand(),
			listCommand(),
			loginCommand(),
			logoutCommand(),
			selectCommand(),
		},
	}
}
