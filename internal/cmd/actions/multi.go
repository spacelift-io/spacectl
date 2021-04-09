package actions

import "github.com/urfave/cli/v2"

// Multi combines multiple CLI actions.
func Multi(steps ...cli.BeforeFunc) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		for _, step := range steps {
			if err := step(ctx); err != nil {
				return err
			}
		}

		return nil
	}
}
