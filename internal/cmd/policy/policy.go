package policy

import (
	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the policyNode command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "policy",
		Usage: "Manage Spacelift policies",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the policies you have access to",
				Flags: []cli.Flag{
					internalCmd.FlagOutputFormat,
					internalCmd.FlagLimit,
					internalCmd.FlagSearch,
				},
				Action: (&listCommand{}).list,
				Before: internalCmd.PerformAllBefore(
					internalCmd.HandleNoColor,
					authenticated.Ensure,
				),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "show",
				Usage: "Shows detailed information about a specific policy",
				Flags: []cli.Flag{
					internalCmd.FlagOutputFormat,
					flagRequiredPolicyID,
				},
				Action:    (&showCommand{}).show,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "samples",
				Usage: "List all policy samples",
				Flags: []cli.Flag{
					internalCmd.FlagOutputFormat,
					internalCmd.FlagNoColor,
					flagRequiredPolicyID,
				},
				Action:    (&samplesCommand{}).list,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "sample",
				Usage: "Inspect one policy sample",
				Flags: []cli.Flag{
					internalCmd.FlagNoColor,
					flagRequiredPolicyID,
					flagRequiredSampleKey,
				},
				Action:    (&sampleCommand{}).show,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "simulate",
				Usage: "Simulate a policy using a sample",
				Flags: []cli.Flag{
					internalCmd.FlagNoColor,
					flagRequiredPolicyID,
					flagSimulationInput,
				},
				Action:    (&simulateCommand{}).simulate,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
		},
	}
}
