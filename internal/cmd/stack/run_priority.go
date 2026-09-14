package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runPriority(ctx context.Context, cliCmd *cli.Command) error {
	if nArgs := cliCmd.NArg(); nArgs != 1 {
		return fmt.Errorf("expected one argument to `priority` but got %d", nArgs)
	}

	preset, err := parseRunPriorityPreset(cliCmd.Args().Get(0))
	if err != nil {
		return err
	}

	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)

	mutation, err := setRunPriorityPreset(ctx, stackID, runID, *preset)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been set to %s priority\n", runID, *preset)
	fmt.Println("The live run can be visited at", authenticated.Client().URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.SetRunPriority.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := logs.NewStackExplorer(stackID, mutation.SetRunPriority.ID).RunFilteredLogs(ctx)
	if err != nil {
		return err
	}

	return terminal.Error()
}

type setRunPriorityPresetMutation struct {
	SetRunPriority struct {
		ID string `graphql:"id"`
	} `graphql:"runPrioritySet(stack: $stackId, run: $runId, preset: $preset)"`
}

func setRunPriorityPreset(ctx context.Context, stackID, runID string, preset structs.RunPriorityPreset) (setRunPriorityPresetMutation, error) {
	var mutation setRunPriorityPresetMutation

	variables := map[string]any{
		"stackId": graphql.ID(stackID),
		"runId":   graphql.ID(runID),
		"preset":  preset,
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return setRunPriorityPresetMutation{}, err
	}

	return mutation, nil
}

func parseRunPriorityPreset(s string) (*structs.RunPriorityPreset, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		return structs.NewRunPriorityPreset("high"), nil
	case "normal":
		return structs.NewRunPriorityPreset("normal"), nil
	case "low":
		return structs.NewRunPriorityPreset("low"), nil
	default:
		return nil, fmt.Errorf("invalid priority level %q (use high, normal or low)", s)
	}
}
