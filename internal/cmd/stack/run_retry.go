package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runRetry(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)

	var mutation struct {
		RunRetry struct {
			ID string `graphql:"id"`
		} `graphql:"runRetry(stack: $stack, run: $run)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"run":   graphql.ID(runID),
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully retried\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.RunRetry.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := logs.NewExplorer(stackID, mutation.RunRetry.ID).RunFilteredLogs(ctx)
	if err != nil {
		return err
	}

	return terminal.Error()
}
