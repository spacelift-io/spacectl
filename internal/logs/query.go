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

func (e *Explorer) runStateLogs(ctx context.Context, state structs.RunState, version int, sink chan<- string, stateTerminal bool) error {
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
		logs, err := e.queryStateLogs(ctx, variables)
		if err != nil {
			return err
		}

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

func (e *Explorer) queryStateLogs(ctx context.Context, variables map[string]any) (*logsQuery, error) {
	type runLogs struct {
		Logs *logsQuery `graphql:"logs(state: $state, token: $token, stateVersion: $stateVersion)"`
	}

	if e.entity == entityModule {
		var query struct {
			Module *struct {
				Run *runLogs `graphql:"run(id: $run)"`
			} `graphql:"module(id: $id)"`
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		if query.Module == nil {
			return nil, fmt.Errorf("%s %q not found", e.entity, e.id)
		}

		if query.Module.Run == nil {
			return nil, fmt.Errorf("run %q in %s %q not found", e.run, e.entity, e.id)
		}

		if query.Module.Run.Logs == nil {
			return nil, fmt.Errorf("logs for run %q in %s %q not found", e.run, e.entity, e.id)
		}

		return query.Module.Run.Logs, nil
	}

	var query struct {
		Stack *struct {
			Run *runLogs `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $id)"`
	}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	if query.Stack == nil {
		return nil, fmt.Errorf("%s %q not found", e.entity, e.id)
	}

	if query.Stack.Run == nil {
		return nil, fmt.Errorf("run %q in %s %q not found", e.run, e.entity, e.id)
	}

	if query.Stack.Run.Logs == nil {
		return nil, fmt.Errorf("logs for run %q in %s %q not found", e.run, e.entity, e.id)
	}

	return query.Stack.Run.Logs, nil
}
