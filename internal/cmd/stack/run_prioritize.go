package stack

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func runPrioritize(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	runID := cliCtx.String(flagRequiredRun.Name)

	mutation, err := setRunPriority(cliCtx, stackID, runID, true)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully prioritized\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.SetRunPriority.ID,
	))

	if !cliCtx.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogsWithAction(cliCtx.Context, stackID, mutation.SetRunPriority.ID, nil)
	if err != nil {
		return err
	}

	return terminal.Error()
}

type setRunPriorityMutation struct {
	SetRunPriority struct {
		ID string `graphql:"id"`
	} `graphql:"runPrioritizeSet(stack: $stackId, run: $runId, prioritize: $prioritize)"`
}

func setRunPriority(cliCtx *cli.Context, stackID, runID string, prioritize bool) (setRunPriorityMutation, error) {
	var mutation setRunPriorityMutation

	variables := map[string]interface{}{
		"stackId":    graphql.ID(stackID),
		"runId":      graphql.ID(runID),
		"prioritize": prioritize,
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return setRunPriorityMutation{}, err
	}

	return mutation, nil
}
