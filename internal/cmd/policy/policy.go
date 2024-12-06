package policy

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the policyNode command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "policy",
		Usage: "Manage Spacelift policies",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the policies you have access to",
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
			{
				Name:  "show",
				Usage: "Shows detailed information about a specific policy",
				Flags: []cli.Flag{
					cmd.FlagOutputFormat,
					flagRequiredPolicyID,
				},
				Action:    (&showCommand{}).show,
				Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Name:  "samples",
				Usage: "List all policy samples",
				Flags: []cli.Flag{
					cmd.FlagOutputFormat,
					cmd.FlagNoColor,
					flagRequiredPolicyID,
				},
				Action:    (&samplesCommand{}).list,
				Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Name:  "sample",
				Usage: "Inspect one policy sample",
				Flags: []cli.Flag{
					cmd.FlagNoColor,
					flagRequiredPolicyID,
					flagRequiredSampleKey,
				},
				Action:    (&sampleCommand{}).show,
				Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Name:  "simulate",
				Usage: "Simulate a policy using a sample",
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
	}
}
