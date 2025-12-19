package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

// actionOnRunState is a function that can be executed on a run state.
//
// It can be used to interact with the run during the log reading,
// for example to confirm a run.
type actionOnRunState func(state structs.RunState, stackID, runID string) error

func runLogs(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd, nil)
	if err != nil {
		return err
	}

	if (!cliCmd.IsSet(flagRun.Name) && !cliCmd.IsSet(flagRunLatest.Name)) ||
		(cliCmd.IsSet(flagRun.Name) && cliCmd.IsSet(flagRunLatest.Name)) {
		return errors.New("you must specify either --run or --run-latest")
	}

	runID := cliCmd.String(flagRun.Name)
	if cliCmd.IsSet(flagRunLatest.Name) {
		type runsQuery struct {
			ID string `graphql:"id"`
		}

		var query struct {
			Stack *struct {
				Runs []runsQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		var before *string
		if err := authenticated.Client().Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return errors.Wrap(err, "failed to query run list")
		}

		if query.Stack == nil {
			return fmt.Errorf("failed to lookup logs with flag --run-latest, stack %q not found", stackID)
		}

		if len(query.Stack.Runs) == 0 {
			return errors.New("failed to lookup logs with flag --run-latest, no runs found")
		}

		runID = query.Stack.Runs[0].ID

		fmt.Println("Using latest run", runID)
	}

	var targetPhase *structs.RunState
	if cliCmd.IsSet(flagPhase.Name) {
		phase := structs.RunState(strings.ToUpper(cliCmd.String(flagPhase.Name)))
		targetPhase = &phase
	}

	_, err = logs.NewExplorer(stackID, runID,
		logs.WithTail(cliCmd.Bool(flagTail.Name)),
		logs.WithTargetPhase(targetPhase),
	).RunFilteredLogs(ctx)

	return err
}
