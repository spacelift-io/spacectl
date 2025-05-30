package stack

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the stack command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "stack",
		Usage: "Manage a Spacelift stack",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Before: authenticated.AttemptAutoLogin,
		Subcommands: []cmd.Command{
			{
				Category: "Run management",
				Name:     "confirm",
				Usage:    "Confirm an unconfirmed tracked run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagRunMetadata,
								flagTail,
							},
							Action:    runConfirm(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "discard",
				Usage:    "Discard an unconfirmed tracked run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
							},
							Action:    runDiscard(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "cancel",
				Usage:    "Cancel a run that hasn't started yet",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
							},
							Action:    runCancel(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "approve",
				Usage:    "Approves a run or task. If no run is specified, the approval will be added to the current stack blocker.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagRunReviewNote,
							},
							Action:    runApprove,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "reject",
				Usage:    "Rejects a run or task. If no run is specified, the rejection will be added to the current stack blocker.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagRunReviewNote,
							},
							Action:    runReject,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "deploy",
				Usage:    "Start a deployment (tracked run)",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagCommitSHA,
								flagRunMetadata,
								flagTail,
								flagAutoConfirm,
								flagRuntimeConfig,
							},
							Action:    runTrigger("TRACKED", "deployment"),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "retry",
				Usage:    "Retry a failed run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
							},
							Action:    runRetry,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "replan",
				Usage:    "Replan an unconfirmed tracked run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
								flagResources,
								flagInteractive,
							},
							Action:    runReplan,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "changes",
				Usage:    "Show a list of changes for a given run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
							},
							Action:    runChanges,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "list",
				Usage: "List the stacks you have access to",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagShowLabels,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
								cmd.FlagLimit,
								cmd.FlagSearch,
							},
							Action:    listStacks(),
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run local preview",
				Name:     "local-preview",
				Usage:    "Start a preview (proposed run) based on the current project. Respects .gitignore and .terraformignore.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagNoFindRepositoryRoot,
								flagProjectRootOnly,
								flagRunMetadata,
								flagNoTail,
								flagNoUpload,
								flagOverrideEnvVars,
								flagOverrideEnvVarsTF,
								flagDisregardGitignore,
								flagPrioritizeRun,
								flagTarget,
							},
							Action:    localPreview(false),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
					{
						EarliestVersion: cmd.SupportedVersion("2.5.0"),
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagNoFindRepositoryRoot,
								flagProjectRootOnly,
								flagRunMetadata,
								flagNoTail,
								flagNoUpload,
								flagOverrideEnvVars,
								flagOverrideEnvVarsTF,
								flagDisregardGitignore,
								flagPrioritizeRun,
								flagTarget,
							},
							Action:    localPreview(true),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "logs",
				Usage:    "Show logs for a particular run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagRunLatest,
							},
							Action:    runLogs,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "preview",
				Usage:    "Start a preview (proposed run)",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagCommitSHA,
								flagRunMetadata,
								flagTail,
								flagRuntimeConfig,
							},
							Action:    runTrigger("PROPOSED", "preview"),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "prioritize",
				Usage:    "Prioritize a run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
							},
							Action:    runPrioritize,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "deprioritize",
				Usage:    "Deprioritize a run",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRequiredRun,
								flagTail,
							},
							Action:    runDeprioritize,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "run",
				Usage: "Manage a stack's runs",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command:         &cli.Command{},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "list",
						Usage: "Lists the runs for a specified stack",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagMaxResults,
										cmd.FlagOutputFormat,
									},
									Action:    runList,
									Before:    authenticated.Ensure,
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
							{
								EarliestVersion: cmd.SupportedVersionLatest,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagMaxResults,
										flagPreviewRuns,
										cmd.FlagOutputFormat,
									},
									Action:    runList,
									Before:    authenticated.Ensure,
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "set-current-commit",
				Usage:    "Set current commit on the stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagRequiredCommitSHA,
							},
							Action:    setCurrentCommit,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "environment",
				Usage: "Manage a stack's environment",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command:         &cli.Command{},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "setvar",
						Usage: "Sets an environment variable.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
										flagEnvironmentWriteOnly,
									},
									Action:    setVar,
									Before:    authenticated.Ensure,
									ArgsUsage: "NAME VALUE",
								},
							},
						},
					},
					{
						Name:  "list",
						Usage: "Lists all the environment variables and mounted files for a stack.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
										cmd.FlagOutputFormat,
									},
									Action: (&listEnvCommand{}).listEnv,
									Before: authenticated.Ensure,
								},
							},
						},
					},
					{
						Name:  "mount",
						Usage: "Mount a file from existing file or STDIN.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
										flagEnvironmentWriteOnly,
									},
									Action:    mountFile,
									Before:    authenticated.Ensure,
									ArgsUsage: "RELATIVE_PATH_TO_MOUNT [FILE_PATH]",
								},
							},
						},
					},
					{
						Name:  "delete",
						Usage: "Deletes an environment variable or mounted file.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
									},
									Action:    deleteEnvironment,
									Before:    authenticated.Ensure,
									ArgsUsage: "NAME",
								},
							},
						},
					},
				},
			},
			{
				Name:  "outputs",
				Usage: "Shows current outputs for a specific stack. Does not show the value of sensitive outputs.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagOutputID,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
							},
							Action:    (&showOutputsStackCommand{}).showOutputs,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "show",
				Usage: "Shows detailed information about a specific stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
							},
							Action:    (&showStackCommand{}).showStack,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "open",
				Usage:    "Open a stack in your browser",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagRun,
								flagIgnoreSubdir,
								flagCurrentBranch,
								flagSearchCount,
							},
							Action:    openCommandInBrowser,
							Before:    authenticated.Ensure,
							ArgsUsage: "COMMAND",
						},
					},
				},
			},
			{
				Category: "Run management",
				Name:     "task",
				Usage:    "Perform a task in a stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
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
				},
			},
			{
				Category: "Stack management",
				Name:     "lock",
				Usage:    "Locks a stack for exclusive use.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagStackLockNote,
							},
							Action:    lock,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "unlock",
				Usage:    "Unlocks a stack.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
							},
							Action:    unlock,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "enable",
				Usage:    "Enable new runs against the stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
							},
							Action:    enable,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "disable",
				Usage:    "Disable new runs against the stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
							},
							Action:    disable,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "sync-commit",
				Usage:    "Syncs the tracked stack commit",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
							},
							Action:    syncCommit,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Stack management",
				Name:     "delete",
				Usage:    "Delete a stack",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagStackID,
								flagDestroyResources,
								flagSkipConfirmation,
							},
							Action:    deleteStack(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "resources",
				Usage: "Manage and view resources for stacks",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command:         &cli.Command{},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "list",
						Usage: "Sets an environment variable.",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
									},
									Action:    resourcesList,
									Before:    authenticated.Ensure,
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
						},
					},
				},
			},
			{
				Name:  "dependencies",
				Usage: "View stack dependencies",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command:         &cli.Command{},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "on",
						Usage: "Get stacks which the provided that depends on",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
										cmd.FlagOutputFormat,
									},
									Action:    dependenciesOn,
									Before:    authenticated.Ensure,
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
						},
					},
					{
						Name:  "off",
						Usage: "Get stacks that depend on the provided stack",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionAll,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagStackID,
										flagRun,
										cmd.FlagOutputFormat,
									},
									Action:    dependenciesOff,
									Before:    authenticated.Ensure,
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
						},
					},
				},
			},
		},
	}
}
