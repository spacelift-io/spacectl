package profile

import (
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
)

func currentCommand() *cli.Command {
	return &cli.Command{
		Name:      "current",
		Usage:     "Outputs your currently selected profile",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(ctx *cli.Context) error {
			currentProfile := manager.Current()

			if currentProfile == nil {
				return errors.New("no account is currently selected")
			}

			fmt.Println(currentProfile.Alias)

			return nil
		},
	}
}
