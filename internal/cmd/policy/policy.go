package policy

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the versioned policy command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "policy",
		Usage: "Manage Spacelift policies",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "list",
				Usage: "List the policies you have access to",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
								cmd.FlagLimit,
								cmd.FlagSearch,
							},
							Action: (&listCommand{}).list,
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
				Name:  "show",
				Usage: "Shows detailed information about a specific policy",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
								flagRequiredPolicyID,
							},
							Action:    (&showCommand{}).show,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "samples",
				Usage: "List all policy samples",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
								flagRequiredPolicyID,
							},
							Action:    (&samplesCommand{}).list,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "sample",
				Usage: "Inspect one policy sample",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagNoColor,
								flagRequiredPolicyID,
								flagRequiredSampleKey,
							},
							Action:    (&sampleCommand{}).show,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "simulate",
				Usage: "Simulate a policy using a sample",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagNoColor,
								flagRequiredPolicyID,
								flagSimulationInput,
							},
							Action:    (&simulateCommand{}).simulate,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}
