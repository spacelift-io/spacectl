package stack

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/cmd/internal/actions"
	"github.com/spacelift-io/spacelift-cli/cmd/internal/authenticated"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "stack",
		Usage:  "Manage a Spacelift stack",
		Flags:  []cli.Flag{flagStackID},
		Before: actions.Multi(authenticated.Ensure, beforeEach),
		Subcommands: []*cli.Command{
			{
				Category: "Run management",
				Name:     "deploy",
				Usage:    "Start a deployment (tracked run)",
				Flags: []cli.Flag{
					flagCommitSHA,
					flagTail,
				},
				Action: runTrigger("TRACKED", "deployment"),
			},
			{
				Category: "Run management",
				Name:     "logs",
				Usage:    "Show logs for a particular run",
				Flags:    []cli.Flag{flagRun},
				Action: func(cliCtx *cli.Context) error {
					return runLogs(context.Background(), stackID, cliCtx.String(flagRun.Name))
				},
			},
			{
				Category: "Run management",
				Name:     "task",
				Usage:    "Perform a task in a workspace",
				Flags: []cli.Flag{
					flagNoInit,
					flagTail,
				},
				Action: taskCommand,
			},
			{
				Category: "Run management",
				Name:     "test",
				Usage:    "Create a proposed (test) run",
				Flags: []cli.Flag{
					flagCommitSHA,
					flagTail,
				},
				Action: runTrigger("PROPOSED", "test run"),
			},
		},
	}
}
