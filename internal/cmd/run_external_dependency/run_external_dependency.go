package run_external_dependency

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the module command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "run-external-dependency",
		Usage: "Manage Spacelift Run external dependencies",
		Subcommands: []*cli.Command{
			{
				Category: "Run external dependency management",
				Name:     "mark-completed",
				Usage:    "Mark Run external dependency as completed",
				Flags: []cli.Flag{
					flagRunExternalDependencyID,
					flagStatus,
				},
				Action:    markRunExternalDependencyAsCompleted,
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}
