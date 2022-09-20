package module

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the module command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "module",
		Usage: "Manage a Spacelift module",
		Subcommands: []*cli.Command{
			{
				Category: "Module management",
				Name:     "create-version",
				Usage:    "Create a new version of a module",
				Flags: []cli.Flag{
					flagModuleID,
					flagCommitSHA,
					flagVersion,
				},
				Action:    createVersion,
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}
