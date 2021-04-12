package profile

import (
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"
)

func currentCommand() *cli.Command {
	return &cli.Command{
		Name:  "current",
		Usage: "Outputs your currently selected profile",

		// Use a space to cause the args usage to not be displayed since the `current` command
		// doesn't accept any arguments
		ArgsUsage: " ",
		Action: func(ctx *cli.Context) error {
			currentProfile, err := manager.Current()
			if err != nil {
				return fmt.Errorf("could not get current profile: %w", err)
			}

			if currentProfile == nil {
				return errors.New("no account is currently selected")
			}

			fmt.Println(currentProfile.Alias)

			return nil
		},
	}
}
