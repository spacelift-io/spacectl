package runexternaldependency

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the run external dependency command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "run-external-dependency",
		Usage: "Manage Spacelift Run external dependencies",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Category: "Run external dependency management",
				Name:     "mark-completed",
				Usage:    "Mark Run external dependency as completed",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagRunExternalDependencyID,
								flagStatus,
							},
							Action:    markRunExternalDependencyAsCompleted,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}
