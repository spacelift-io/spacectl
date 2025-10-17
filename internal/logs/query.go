package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

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
		"stateVersion": graphql.Int(version), //nolint: gosec
	}

	var backOff time.Duration

	for {
		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
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
