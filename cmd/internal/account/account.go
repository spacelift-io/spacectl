package account

import "github.com/urfave/cli/v2"

// Command encapsulates the account command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:   "account",
		Usage:  "Manage Spacelift accounts",
		Before: beforeEach,
		Subcommands: []*cli.Command{
			loginCommand(),
			logoutCommand(),
			selectCommand(),
		},
	}
}
