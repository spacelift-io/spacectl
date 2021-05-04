package stack

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/actions"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the stack command subtree.
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
				Category: "Run local preview",
				Name:     "local-preview",
				Usage:    "Start a preview (proposed run) based on the current directory. Respects .gitignore and .terraformignore.",
				Flags: []cli.Flag{
					flagNoTail,
				},
				Action: localPreview(),
			},
			{
				Category: "Run management",
				Name:     "logs",
				Usage:    "Show logs for a particular run",
				Flags:    []cli.Flag{flagRun},
				Action: func(cliCtx *cli.Context) error {
					_, err := runLogs(context.Background(), stackID, cliCtx.String(flagRun.Name))
					return err
				},
			},
			{
				Category: "Run management",
				Name:     "preview",
				Usage:    "Start a preview (proposed run)",
				Flags: []cli.Flag{
					flagCommitSHA,
					flagTail,
				},
				Action: runTrigger("PROPOSED", "preview"),
			},
			{
				Category: "Stack management",
				Name:     "set-current-commit",
				Usage:    "Set current commit on the stack",
				Flags:    []cli.Flag{flagRequiredCommitSHA},
				Action:   setCurrentCommit,
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
		},
	}
}
