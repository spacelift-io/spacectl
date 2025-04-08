package actions

import (
	"context"

	"github.com/urfave/cli/v3"
)

// Multi combines multiple CLI actions.
func Multi(steps ...cli.BeforeFunc) cli.BeforeFunc {
	return func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		for _, step := range steps {
			if ctx, err := step(ctx, cmd); err != nil {
				return ctx, err
			}
		}

		return ctx, nil
	}
}
