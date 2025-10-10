package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runCancel() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd)
		if err != nil {
			return err
		}

		var mutation struct {
			RunDiscard struct {
				ID string `graphql:"id"`
			} `graphql:"runCancel(stack: $stack, run: $run)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(cliCmd.String(flagRequiredRun.Name)),
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("You have successfully canceled the run")

		fmt.Println("The run can be visited at", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunDiscard.ID,
		))

		if !cliCmd.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := logs.NewExplorer(stackID, mutation.RunDiscard.ID).RunFilteredLogs(ctx)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
