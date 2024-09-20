package blueprint

import (
	"fmt"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
	"math"
)

// Command encapsulates the blueprintNode command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "blueprint",
		Usage: "Manage a Spacelift blueprints",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the blueprints you have access to",
				Flags: []cli.Flag{
					flagShowLabels,
					cmd.FlagOutputFormat,
					cmd.FlagNoColor,
					flagLimit,
					flagSearch,
				},
				Action: listBlueprints(),
				Before: cmd.PerformAllBefore(
					cmd.HandleNoColor,
					authenticated.Ensure,
					validateLimit,
					validateSearch,
				),
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}

func validateLimit(cliCtx *cli.Context) error {
	if cliCtx.IsSet(flagLimit.Name) {
		if cliCtx.Uint(flagLimit.Name) == 0 {
			return fmt.Errorf("limit must be greater than 0")
		}

		if cliCtx.Uint(flagLimit.Name) >= math.MaxInt32 {
			return fmt.Errorf("limit must be less than %d", math.MaxInt32)
		}
	}

	return nil
}

func validateSearch(cliCtx *cli.Context) error {
	if cliCtx.IsSet(flagSearch.Name) {
		if cliCtx.String(flagSearch.Name) == "" {
			return fmt.Errorf("search must be non-empty")
		}

	}

	return nil
}
