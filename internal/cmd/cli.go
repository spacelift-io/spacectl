package cmd

import (
	"context"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

// PerformAllBefore wraps all the specified BeforeFuncs into a single BeforeFunc.
func PerformAllBefore(actions ...cli.BeforeFunc) cli.BeforeFunc {
	return func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		for i := range actions {
			action := actions[i]
			if _, err := action(ctx, cmd); err != nil {
				return ctx, err
			}
		}

		return ctx, nil
	}
}

// HandleNoColor handles FlagNoColor to disable console coloring.
func HandleNoColor(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	noColor := cmd.Bool(FlagNoColor.Name)
	isTerminal := isatty.IsTerminal(os.Stdout.Fd())

	if noColor || !isTerminal {
		pterm.DisableColor()
	}

	return ctx, nil
}
