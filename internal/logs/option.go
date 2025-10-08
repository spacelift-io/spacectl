package logs

import "github.com/spacelift-io/spacectl/client/structs"

// Option is a functional option for configuring an Explorer
type Option func(*Explorer)

type ActionOnRunState func(state structs.RunState, stackID, runID string) error

// WithActionOnRunState sets an action to be executed on each run state
func WithActionOnRunState(acFn ActionOnRunState) Option {
	return func(e *Explorer) {
		e.acFn = acFn
	}
}

// WithTargetPhase sets the target phase to filter logs
func WithTargetPhase(phase *structs.RunState) Option {
	return func(e *Explorer) {
		e.targetPhase = phase
	}
}

// WithTail sets whether to tail the logs
func WithTail(tail bool) Option {
	return func(e *Explorer) {
		e.tail = tail
	}
}
