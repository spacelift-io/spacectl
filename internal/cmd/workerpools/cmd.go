package workerpools

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the workerpool command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "workerpool",
		Usage: "Manages workerpools and their workers.",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "Lists all worker pools.",
				Flags: []cli.Flag{
					cmd.FlagOutputFormat,
				},
				Action: (&listPoolsCommand{}).listPools,
				Before: authenticated.Ensure,
			},
			{
				Name:  "worker",
				Usage: "Contains commands for managing workers within a pool.",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "Lists all workers of a workerpool.",
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
	}
}
