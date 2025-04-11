package workerpools

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the workerpool command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "workerpool",
		Usage: "Manages workerpools and their workers.",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "list",
				Usage: "Lists all worker pools.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
							},
							Action: (&listPoolsCommand{}).listPools,
							Before: authenticated.Ensure,
						},
					},
				},
			},
			{
				Name:  "watch",
				Usage: "Starts an interactive watcher for a worker pool",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Action: watch,
							Before: authenticated.Ensure,
						},
					},
				},
			},
			{
				Name:  "worker",
				Usage: "Contains commands for managing workers within a pool.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command:         &cli.Command{},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "cycle",
						Usage: "Sends a kill signal to all workers in a workerpool.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagPoolIDNamed,
									},
									Action: (&cycleWorkersCommand{}).cycleWorkers,
									Before: authenticated.Ensure,
								},
							},
						},
					},
					{
						Name:  "list",
						Usage: "Lists all workers of a workerpool.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagPoolIDNamed,
										cmd.FlagOutputFormat,
									},
									Action: (&listWorkersCommand{}).listWorkers,
									Before: authenticated.Ensure,
								},
							},
						},
					},
					{
						Name:  "drain",
						Usage: "Drains a worker.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagWorkerID,
										flagPoolIDNamed,
										flagWaitUntilDrained,
									},
									Action: (&drainWorkerCommand{}).drainWorker,
									Before: authenticated.Ensure,
								},
							},
						},
					},
					{
						Name:  "undrain",
						Usage: "Undrains a worker.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagWorkerID,
										flagPoolIDNamed,
									},
									Action: (&undrainWorkerCommand{}).undrainWorker,
									Before: authenticated.Ensure,
								},
							},
						},
					},
				},
			},
		},
	}
}
