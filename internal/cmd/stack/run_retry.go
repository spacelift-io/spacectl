package stack

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func runRetry(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	runID := cliCtx.String(flagRequiredRun.Name)

	var mutation struct {
		RunRetry struct {
			ID string `graphql:"id"`
		} `graphql:"runRetry(stack: $stack, run: $run)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"run":   graphql.ID(runID),
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully retried\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.RunRetry.ID,
	))

	if !cliCtx.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogsWithAction(cliCtx.Context, stackID, mutation.RunRetry.ID, nil)
	if err != nil {
		return err
	}

	return terminal.Error()
}
