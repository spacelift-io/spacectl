package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runPrioritize(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)

	mutation, err := setRunPriority(ctx, stackID, runID, true)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully prioritized\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.SetRunPriority.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogsWithAction(ctx, stackID, mutation.SetRunPriority.ID, nil)
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

func setRunPriority(ctx context.Context, stackID, runID string, prioritize bool) (setRunPriorityMutation, error) {
	var mutation setRunPriorityMutation

	variables := map[string]interface{}{
		"stackId":    graphql.ID(stackID),
		"runId":      graphql.ID(runID),
		"prioritize": graphql.Boolean(prioritize),
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return setRunPriorityMutation{}, err
	}

	return mutation, nil
}
