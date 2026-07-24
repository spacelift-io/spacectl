package module

import (
	"context"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func moduleLogs(ctx context.Context, cliCmd *cli.Command) error {
	moduleID := cliCmd.String(flagModuleID.Name)
	runID := cliCmd.String(flagRun.Name)

	var targetPhase *structs.RunState
	if cliCmd.IsSet(flagPhase.Name) {
		phase := structs.RunState(strings.ToUpper(cliCmd.String(flagPhase.Name)))
		targetPhase = &phase
	}

	_, err := logs.NewExplorer(moduleID, runID,
		logs.WithModule(),
		logs.WithTail(cliCmd.Bool(flagTail.Name)),
		logs.WithTargetPhase(targetPhase),
	).RunFilteredLogs(ctx)

	return err
}
