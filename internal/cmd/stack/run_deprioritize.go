package stack

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runDeprioritize(ctx context.Context, cmd *cli.Command) error {
	stackID, err := getStackID(ctx, cmd)
	if err != nil {
		return err
	}
	runID := cmd.String(flagRequiredRun.Name)

	mutation, err := setRunPriority(ctx, stackID, runID, false)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully deprioritized\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.SetRunPriority.ID,
	))

	if !cmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogsWithAction(ctx, stackID, mutation.SetRunPriority.ID, nil)
	if err != nil {
		return err
	}

	return terminal.Error()
}
