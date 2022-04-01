package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

// PerformAllBefore wraps all the specified BeforeFuncs into a single BeforeFunc.
func PerformAllBefore(actions ...cli.BeforeFunc) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		for i := range actions {
			action := actions[i]
			if err := action(ctx); err != nil {
				return err
			}
		}

		return nil
	}
}

// HandleNoColor handles FlagNoColor to disable console coloring.
func HandleNoColor(ctx *cli.Context) error {
	noColor := ctx.Bool(FlagNoColor.Name)
	isTerminal := isatty.IsTerminal(os.Stdout.Fd())

	if noColor || !isTerminal {
		pterm.DisableColor()
	}

	return nil
}
