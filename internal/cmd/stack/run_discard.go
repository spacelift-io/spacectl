package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runDiscard() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stackID, err := getStackID(ctx, cmd)
		if err != nil {
			return err
		}

		var mutation struct {
			RunDiscard struct {
				ID string `graphql:"id"`
			} `graphql:"runDiscard(stack: $stack, run: $run)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(cmd.String(flagRequiredRun.Name)),
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("You have successfully discarded a deployment")

		fmt.Println("The run can be visited at", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunDiscard.ID,
		))

		if !cmd.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := runLogsWithAction(ctx, stackID, mutation.RunDiscard.ID, nil)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
