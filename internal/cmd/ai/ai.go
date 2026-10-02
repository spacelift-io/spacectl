// Package ai contains commands for working with Spacelift Intelligence features.
package ai

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var (
	flagTranscriptOffset = &cli.IntFlag{
		Name:  "offset",
		Usage: "[Optional] Index of the first message to include in the transcript",
	}

	flagTranscriptLimit = &cli.IntFlag{
		Name:  "limit",
		Usage: "[Optional] Maximum number of messages to include in the transcript (0 means all)",
	}
)

// Command returns the ai command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "ai",
		Usage: "Work with Spacelift Intelligence",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionLatest,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "sessions",
				Usage: "Manage your Infra Assistant sessions",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionLatest,
						Command: &cli.Command{
							Description: "Each session belongs to the user who created it, so the profile must be logged in as you (spacectl profile login, web browser option). API key profiles can't read Infra Assistant sessions.",
						},
					},
				},
				Subcommands: []cmd.Command{
					{
						Name:  "list",
						Usage: "List your Infra Assistant sessions, most recently updated first",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionLatest,
								Command: &cli.Command{
									Flags: []cli.Flag{
										cmd.FlagOutputFormat,
										cmd.FlagNoColor,
										cmd.FlagLimit,
										cmd.FlagSearch,
									},
									Action: listSessions,
									Before: cmd.PerformAllBefore(
										cmd.HandleNoColor,
										authenticated.Ensure,
									),
									ArgsUsage: cmd.EmptyArgsUsage,
								},
							},
						},
					},
					{
						Name:  "get",
						Usage: "Print the markdown transcript of an Infra Assistant session to stdout",
						Versions: []cmd.VersionedCommand{
							{
								EarliestVersion: cmd.SupportedVersionLatest,
								Command: &cli.Command{
									Flags: []cli.Flag{
										flagTranscriptOffset,
										flagTranscriptLimit,
									},
									Action:    getSession,
									Before:    authenticated.Ensure,
									ArgsUsage: "<session-id>",
								},
							},
						},
					},
				},
			},
		},
	}
}
