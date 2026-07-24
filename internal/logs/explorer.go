package logs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Explorer allows you to explore run logs, either for a stack or a module.
//
// It's a single use object, which should be thrown away after use.
type Explorer struct {
	id     string
	run    string
	tail   bool
	entity entity

	acFn               ActionOnRunState
	targetPhase        *structs.RunState
	targetPhaseReached bool

	backoff time.Duration
}

// NewExplorer creates a new Explorer with the given options.
// By default the explorer explores a stack's run and always tails the logs;
// use WithModule to explore a module's run instead.
func NewExplorer(id, run string, opts ...Option) *Explorer {
	e := &Explorer{
		id:      id,
		run:     run,
		tail:    true,
		backoff: 0,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// RunFilteredLogs runs the explorer, printing filtered logs to stdout.
func (e *Explorer) RunFilteredLogs(ctx context.Context) (terminal *structs.RunStateTransition, err error) {
	lines := make(chan string)

	go func() {
		terminal, err = e.RunFilteredStates(ctx, lines)
		close(lines)
	}()

	for line := range lines {
		fmt.Print(line)
	}

	return
}

// RunFilteredStates runs the explorer, sending filtered logs to the given sink channel.
//
// Usually you want to use RunFilteredLogs instead.
func (e *Explorer) RunFilteredStates(ctx context.Context, sink chan<- string) (*structs.RunStateTransition, error) {
	reportedStates := make(map[structs.RunState]struct{})

	for {
		history, err := e.getHistory(ctx)
		if err != nil {
			return nil, err
		}

		transition, ok, err := e.processHistory(ctx, sink, history, reportedStates)
		if err != nil {
			return nil, err
		}

		if !ok {
			return transition, nil
		}

		time.Sleep(e.backoff * time.Second)

		if e.backoff < 5 {
			e.backoff++
		}
	}
}

func (e *Explorer) getHistory(ctx context.Context) ([]structs.RunStateTransition, error) {
	type runHistory struct {
		History []structs.RunStateTransition `graphql:"history"`
	}

	variables := map[string]any{
		"id":  graphql.ID(e.id),
		"run": graphql.ID(e.run),
	}

	if e.entity == entityModule {
		var query struct {
			Module *struct {
				Run *runHistory `graphql:"run(id: $run)"`
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

		return query.Module.Run.History, nil
	}

	var query struct {
		Stack *struct {
			Run *runHistory `graphql:"run(id: $run)"`
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

	return query.Stack.Run.History, nil
}

func (e *Explorer) processHistory(ctx context.Context, sink chan<- string, history []structs.RunStateTransition, reportedStates map[structs.RunState]struct{}) (*structs.RunStateTransition, bool, error) {
	var transition structs.RunStateTransition

	for _, transition = range slices.Backward(history) {
		if _, ok := reportedStates[transition.State]; ok {
			continue
		}
		e.backoff = 0
		reportedStates[transition.State] = struct{}{}

		skip, terminal, err := e.processTargetPhase(&transition, sink)
		if err != nil {
			return nil, false, err
		}

		if skip {
			if terminal {
				return &transition, false, nil
			}
			continue
		}

		e.print(&transition, sink)

		terminal, err = e.processTransition(ctx, &transition, sink)
		if err != nil {
			return nil, false, err
		}

		if terminal {
			return &transition, false, nil
		}

	}

	if !e.tail {
		return &transition, false, nil
	}

	return &transition, true, nil
}

func (e *Explorer) processTargetPhase(transition *structs.RunStateTransition, sink chan<- string) (skip bool, terminal bool, err error) {
	if e.targetPhase == nil {
		return false, false, nil
	}

	if transition.State == *e.targetPhase {
		e.targetPhaseReached = true
		return false, false, nil
	}

	if transition.Terminal && !e.targetPhaseReached {
		sink <- fmt.Sprintf("Run completed without reaching phase %s\n", *e.targetPhase)
		return false, false, errors.New("filtering failed")
	}

	return true, transition.Terminal, nil
}

func (e *Explorer) processTransition(ctx context.Context, transition *structs.RunStateTransition, sink chan<- string) (bool, error) {
	if transition.HasLogs {
		if err := e.runStateLogs(ctx, transition.State, transition.StateVersion, sink, transition.Terminal); err != nil {
			return false, err
		}
	}

	if err := e.actionFunc(transition.State); err != nil {
		return false, err
	}

	return transition.Terminal, nil
}

func (e *Explorer) print(transition *structs.RunStateTransition, sink chan<- string) {
	if e.targetPhase != nil {
		return
	}

	sink <- fmt.Sprintf(`
-----------------
%s
-----------------

`, transition.About())

}

func (e *Explorer) actionFunc(state structs.RunState) error {
	if e.acFn == nil {
		return nil
	}

	if err := e.acFn(state, e.id, e.run); err != nil {
		return fmt.Errorf("failed to execute action on run state: %w", err)
	}

	return nil
}
