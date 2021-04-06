package account

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func selectCommand() *cli.Command {
	return &cli.Command{
		Name:  "select",
		Usage: "Select one of existing Spacelift accounts",
		Action: func(*cli.Context) error {
			if _, err := os.Stat(aliasPath); err != nil {
				return fmt.Errorf("could not select account %s: %w", accountAlias, err)
			}

			return setCurrentAccount()
		},
	}
}
