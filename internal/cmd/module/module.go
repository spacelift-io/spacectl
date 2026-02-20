package module

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func Command() cmd.Command {
	return cmd.Command{
		Name:  "module",
		Usage: "Manage a Spacelift module",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Category: "Module management",
				Name:     "create-version",
				Usage:    "Create a new version of a module",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
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
				},
			},
			{
				Category: "Module management",
				Name:     "delete-version",
				Usage:    "Delete a version of a module",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagModuleID,
								flagVersionID,
							},
							Action:    deleteVersion,
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Module management",
				Name:     "local-preview",
				Usage:    "Start a preview (proposed version) based on the current project. Respects .gitignore and .terraformignore.",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagModuleID,
								flagNoFindRepositoryRoot,
								flagNoUpload,
								flagRunMetadata,
								flagDisregardGitignore,
								flagWithGitDir,
								flagTests,
								flagNoAnimation,
							},
							Action:    localPreviewFunc(false),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
					{
						EarliestVersion: cmd.SupportedVersion("2.5.0"),
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagModuleID,
								flagNoFindRepositoryRoot,
								flagNoUpload,
								flagRunMetadata,
								flagDisregardGitignore,
								flagWithGitDir,
								flagTests,
								flagNoAnimation,
							},
							Action:    localPreviewFunc(true),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Module management",
				Name:     "list",
				Usage:    "List all modules available and their current version",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
								cmd.FlagLimit,
								cmd.FlagSearch,
							},
							Action:    listModules(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Module management",
				Name:     "list-versions",
				Usage:    "List 20 latest non failed versions for a module",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagModuleID,
								cmd.FlagOutputFormat,
							},
							Action:    listVersions(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}
