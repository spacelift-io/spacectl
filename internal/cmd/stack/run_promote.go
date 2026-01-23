package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runPromote() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd)
		if err != nil {
			return err
		}

		var mutation struct {
			RunPromote struct {
				ID string `graphql:"id"`
			} `graphql:"runPromote(stack: $stack, run: $run)"`
		}

		variables := map[string]any{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(cliCmd.String(flagRequiredRun.Name)),
		}

		if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("You have successfully promoted the run to a tracked run")

		fmt.Println("The live run can be visited at", authenticated.Client().URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunPromote.ID,
		))

		if !cliCmd.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := logs.NewExplorer(stackID, mutation.RunPromote.ID).RunFilteredLogs(ctx)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
