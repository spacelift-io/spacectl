package stack

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runDeprioritize(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)

	mutation, err := setRunPriority(ctx, stackID, runID, false)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully deprioritized\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client().URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.SetRunPriority.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := logs.NewExplorer(stackID, mutation.SetRunPriority.ID).RunFilteredLogs(ctx)
	if err != nil {
		return err
	}

	return terminal.Error()
}
