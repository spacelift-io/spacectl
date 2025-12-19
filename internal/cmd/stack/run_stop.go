package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runStop() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd, nil)
		if err != nil {
			return err
		}

		var mutation struct {
			RunStop struct {
				ID string `graphql:"id"`
			} `graphql:"runStop(stack: $stack, run: $run, note: $note)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(cliCmd.String(flagRequiredRun.Name)),
			"note":  graphql.String("Stopped by spacectl"),
		}

		if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("You have successfully attempted to stop the run")

		fmt.Println("The run can be visited at", authenticated.Client().URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunStop.ID,
		))

		if !cliCmd.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := logs.NewExplorer(stackID, mutation.RunStop.ID).RunFilteredLogs(ctx)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
