package blueprint

import (
	"context"
	"fmt"
	"math"

	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the blueprintNode command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "blueprint",
		Usage: "Manage a Spacelift blueprints",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the blueprints you have access to",
				Flags: []cli.Flag{
					internalCmd.FlagShowLabels,
					internalCmd.FlagOutputFormat,
					internalCmd.FlagNoColor,
					internalCmd.FlagLimit,
					internalCmd.FlagSearch,
				},
				Action: listBlueprints(),
				Before: internalCmd.PerformAllBefore(
					internalCmd.HandleNoColor,
					authenticated.Ensure,
					validateLimit,
					validateSearch,
				),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "show",
				Usage: "Shows detailed information about a specific blueprint",
				Flags: []cli.Flag{
					flagRequiredBlueprintID,
					internalCmd.FlagOutputFormat,
					internalCmd.FlagNoColor,
				},
				Action:    (&showCommand{}).show,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
			{
				Name:  "deploy",
				Usage: "Deploy a stack from the blueprint",
				Flags: []cli.Flag{
					flagRequiredBlueprintID,
					internalCmd.FlagNoColor,
				},
				Action:    (&deployCommand{}).deploy,
				Before:    internalCmd.PerformAllBefore(internalCmd.HandleNoColor, authenticated.Ensure),
				ArgsUsage: internalCmd.EmptyArgsUsage,
			},
		},
	}
}

func validateLimit(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if cmd.IsSet(internalCmd.FlagLimit.Name) {
		if cmd.Uint(internalCmd.FlagLimit.Name) == 0 {
			return ctx, fmt.Errorf("limit must be greater than 0")
		}

		if cmd.Uint(internalCmd.FlagLimit.Name) >= math.MaxInt32 {
			return ctx, fmt.Errorf("limit must be less than %d", math.MaxInt32)
		}
	}

	return ctx, nil
}

func validateSearch(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if cmd.IsSet(internalCmd.FlagSearch.Name) {
		if cmd.String(internalCmd.FlagSearch.Name) == "" {
			return ctx, fmt.Errorf("search must be non-empty")
		}

	}

	return ctx, nil
}
