package stack

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runLogs(ctx context.Context, stack, run string) (terminal *structs.RunStateTransition, err error) {
	lines := make(chan string)

	go func() {
		terminal, err = runStates(ctx, stack, run, lines)
		close(lines)
	}()

	for line := range lines {
		fmt.Print(line)
	}

	return
}

func runStates(ctx context.Context, stack, run string, sink chan<- string) (*structs.RunStateTransition, error) {
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
				if err := runStateLogs(ctx, stack, run, transition.State, transition.StateVersion, sink); err != nil {
					return nil, err
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

func runStateLogs(ctx context.Context, stack, run string, state structs.RunState, version int, sink chan<- string) error {
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
					NextToken *graphql.String `graphql:"nextToken"`
				} `graphql:"logs(state: $state, token: $token, stateVersion: $stateVersion)"`
			} `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $stack)"`
	}

	var token *graphql.String
	variables := map[string]interface{}{
		"stack":        graphql.ID(stack),
		"run":          graphql.ID(run),
		"state":        state,
		"token":        token,
		"stateVersion": graphql.Int(version),
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

		if logs.Finished {
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
