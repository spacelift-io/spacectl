package stack

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the stack command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "stack",
		Usage: "Manage a Spacelift stack",
		Subcommands: []*cli.Command{
			{
				Category: "Run management",
				Name:     "confirm",
				Usage:    "Confirm an unconfirmed tracked run",
				Flags: []cli.Flag{
					flagStackID,
					flagRun,
					flagRunMetadata,
					flagTail,
				},
				Action:    runConfirm(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Run management",
				Name:     "deploy",
				Usage:    "Start a deployment (tracked run)",
				Flags: []cli.Flag{
					flagStackID,
					flagCommitSHA,
					flagRunMetadata,
					flagTail,
				},
				Action:    runTrigger("TRACKED", "deployment"),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Name:  "list",
				Usage: "List the stacks you have access to",
				Flags: []cli.Flag{
					cmd.FlagOutputFormat,
				},
				Action:    listStacks(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Run local preview",
				Name:     "local-preview",
				Usage:    "Start a preview (proposed run) based on the current project. Respects .gitignore and .terraformignore.",
				Flags: []cli.Flag{
					flagStackID,
					flagNoFindRepositoryRoot,
					flagRunMetadata,
					flagNoTail,
				},
				Action:    localPreview(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Run management",
				Name:     "logs",
				Usage:    "Show logs for a particular run",
				Flags: []cli.Flag{
					flagStackID,
					flagRun,
				},
				Action: func(cliCtx *cli.Context) error {
					stackID := cliCtx.String(flagStackID.Name)
					_, err := runLogs(context.Background(), stackID, cliCtx.String(flagRun.Name))
					return err
				},
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Run management",
				Name:     "preview",
				Usage:    "Start a preview (proposed run)",
				Flags: []cli.Flag{
					flagStackID,
					flagCommitSHA,
					flagRunMetadata,
					flagTail,
				},
				Action:    runTrigger("PROPOSED", "preview"),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Stack management",
				Name:     "set-current-commit",
				Usage:    "Set current commit on the stack",
				Flags: []cli.Flag{
					flagStackID,
					flagRequiredCommitSHA,
				},
				Action:    setCurrentCommit,
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Run management",
				Name:     "task",
				Usage:    "Perform a task in a workspace",
				Flags: []cli.Flag{
					flagStackID,
					flagNoInit,
					flagRunMetadata,
					flagTail,
				},
				Action:    taskCommand,
				Before:    authenticated.Ensure,
				ArgsUsage: "COMMAND",
			},
		},
	}
}
