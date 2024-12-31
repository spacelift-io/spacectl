package stack

import (
	"context"
	"fmt"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// actionOnRunState is a function that can be executed on a run state.
//
// It can be used to interact with the run during the log reading,
// for example to confirm a run.
type actionOnRunState func(state structs.RunState, stackID, runID string) error

func runLogs(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	if (!cliCtx.IsSet(flagRun.Name) && !cliCtx.IsSet(flagRunLatest.Name)) ||
		(cliCtx.IsSet(flagRun.Name) && cliCtx.IsSet(flagRunLatest.Name)) {
		return errors.New("you must specify either --run or --run-latest")
	}

	runID := cliCtx.String(flagRun.Name)
	if cliCtx.IsSet(flagRunLatest.Name) {
		type runsQuery struct {
			ID string `graphql:"id"`
		}

		var query struct {
			Stack *struct {
				Runs []runsQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		var before *string
		if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
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

	_, err = runLogsWithAction(context.Background(), stackID, runID, nil)
	return err
}

func runLogsWithAction(ctx context.Context, stack, run string, acFn actionOnRunState) (terminal *structs.RunStateTransition, err error) {
	lines := make(chan string)

	go func() {
		terminal, err = runStates(ctx, stack, run, lines, acFn)
		close(lines)
	}()

	for line := range lines {
		fmt.Print(line)
	}

	return
}

func runStates(ctx context.Context, stack, run string, sink chan<- string, acFn actionOnRunState) (*structs.RunStateTransition, error) {
	var query struct {
		Stack *struct {
			Run *struct {
				History []structs.RunStateTransition `graphql:"history"`
			} `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $stack)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stack),
		"run":   graphql.ID(run),
	}

	reportedStates := make(map[structs.RunState]struct{})

	var backoff = time.Duration(0)

	for {
		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		if query.Stack == nil {
			return nil, fmt.Errorf("stack %q not found", stack)
		}

		if query.Stack.Run == nil {
			return nil, fmt.Errorf("run %q in stack %q not found", run, stack)
		}

		history := query.Stack.Run.History

		for index := range history {
			// Unlike the GUI, we go earliest first.
			transition := history[len(history)-index-1]

			if _, ok := reportedStates[transition.State]; ok {
				continue
			}
			backoff = 0
			reportedStates[transition.State] = struct{}{}

			fmt.Println("")
			fmt.Println("-----------------")
			fmt.Println(transition.About())
			fmt.Println("-----------------")
			fmt.Println("")

			if transition.HasLogs {
				if err := runStateLogs(ctx, stack, run, transition.State, transition.StateVersion, sink, transition.Terminal); err != nil {
					return nil, err
				}
			}

			if acFn != nil {
				if err := acFn(transition.State, stack, run); err != nil {
					return nil, fmt.Errorf("failed to execute action on run state: %w", err)
				}
			}

			if transition.Terminal {
				return &transition, nil
			}
		}

		time.Sleep(backoff * time.Second)

		if backoff < 5 {
			backoff++
		}
	}
}

func runStateLogs(ctx context.Context, stack, run string, state structs.RunState, version int, sink chan<- string, stateTerminal bool) error {
	var query struct {
		Stack *struct {
			Run *struct {
				Logs *struct {
					Exists   bool `graphql:"exists"`
					Finished bool `graphql:"finished"`
					HasMore  bool `graphql:"hasMore"`
					Messages []struct {
						Body string `graphql:"message"`
					} `graphql:"messages"`
					NextToken *string `graphql:"nextToken"`
				} `graphql:"logs(state: $state, token: $token, stateVersion: $stateVersion)"`
			} `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $stack)"`
	}

	var token *string
	variables := map[string]interface{}{
		"stack":        graphql.ID(stack),
		"run":          graphql.ID(run),
		"state":        state,
		"token":        token,
		"stateVersion": version,
	}

	var backOff time.Duration

	for {
		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return err
		}

		if query.Stack == nil {
			return fmt.Errorf("stack %q not found", stack)
		}

		if query.Stack.Run == nil {
			return fmt.Errorf("run %q in stack %q not found", run, stack)
		}

		if query.Stack.Run.Logs == nil {
			return fmt.Errorf("logs for run %q in stack %q not found", run, stack)
		}

		logs := query.Stack.Run.Logs
		variables["token"] = logs.NextToken

		for _, message := range logs.Messages {
			sink <- message.Body
		}

		if logs.Finished || (!logs.HasMore && stateTerminal) {
			break
		}

		if logs.HasMore {
			backOff = 0
		} else {
			backOff++
		}

		time.Sleep(backOff * time.Second)
	}

	return nil
}
