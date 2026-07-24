package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type logsQuery struct {
	Exists   bool `graphql:"exists"`
	Finished bool `graphql:"finished"`
	HasMore  bool `graphql:"hasMore"`
	Messages []struct {
		Body string `graphql:"message"`
	} `graphql:"messages"`
	NextToken *graphql.String `graphql:"nextToken"`
}

func queryRun[T any](ctx context.Context, e *Explorer, variables map[string]any) (*T, error) {
	var run *T

	if e.entity == entityModule {
		var query struct {
			Module *struct {
				Run *T `graphql:"run(id: $run)"`
			} `graphql:"module(id: $id)"`
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		if query.Module == nil {
			return nil, fmt.Errorf("%s %q not found", e.label(), e.id)
		}

		run = query.Module.Run
	} else {
		var query struct {
			Stack *struct {
				Run *T `graphql:"run(id: $run)"`
			} `graphql:"stack(id: $id)"`
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		if query.Stack == nil {
			return nil, fmt.Errorf("%s %q not found", e.label(), e.id)
		}

		run = query.Stack.Run
	}

	if run == nil {
		return nil, fmt.Errorf("run %q in %s %q not found", e.run, e.label(), e.id)
	}

	return run, nil
}

func (e *Explorer) runStateLogs(ctx context.Context, state structs.RunState, version int, sink chan<- string, stateTerminal bool) error {
	type runLogs struct {
		Logs *logsQuery `graphql:"logs(state: $state, token: $token, stateVersion: $stateVersion)"`
	}

	var token *graphql.String
	variables := map[string]any{
		"id":           graphql.ID(e.id),
		"run":          graphql.ID(e.run),
		"state":        state,
		"token":        token,
		"stateVersion": graphql.Int(version), //nolint: gosec
	}

	var backOff time.Duration

	for {
		run, err := queryRun[runLogs](ctx, e, variables)
		if err != nil {
			return err
		}

		if run.Logs == nil {
			return fmt.Errorf("logs for run %q in %s %q not found", e.run, e.label(), e.id)
		}

		logs := run.Logs
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
