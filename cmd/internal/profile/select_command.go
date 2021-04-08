package profile

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:   "select",
		Usage:  "Select one of existing Spacelift profile",
		Before: getAlias,
		Action: func(*cli.Context) error {
			if _, err := os.Stat(aliasPath); err != nil {
				return fmt.Errorf("could not select profile %s: %w", profileAlias, err)
			}

			return setCurrentProfile()
		},
	}
}
