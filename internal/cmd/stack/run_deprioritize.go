package stack

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runDeprioritize(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	runID := cliCtx.String(flagRequiredRun.Name)

	mutation, err := setRunPriority(cliCtx.Context, stackID, runID, false)
	if err != nil {
		return err
	}

	fmt.Printf("Run ID %q has been successfully deprioritized\n", runID)
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
