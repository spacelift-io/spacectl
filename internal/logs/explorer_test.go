package logs

import (
	"strings"
	"testing"

	"github.com/spacelift-io/spacectl/client/structs"
)

func TestProcessTargetPhase(t *testing.T) {
	type testCase struct {
		name               string
		targetPhase        *structs.RunState
		targetPhaseReached bool
		transition         structs.RunStateTransition
		wantSkip           bool
		wantTerminal       bool
		wantError          bool
		wantErrorMsg       string
	}

	planningState := structs.RunState("PLANNING")
	applyingState := structs.RunState("APPLYING")
	finishedState := structs.RunState("FINISHED")

	testCases := []testCase{
		{
			name:         "no target phase set",
			targetPhase:  nil,
			transition:   structs.RunStateTransition{State: planningState, Terminal: false},
			wantSkip:     false,
			wantTerminal: false,
			wantError:    false,
		},
		{
			name:               "target phase reached",
			targetPhase:        &planningState,
			targetPhaseReached: false,
			transition:         structs.RunStateTransition{State: planningState, Terminal: false},
			wantSkip:           false,
			wantTerminal:       false,
			wantError:          false,
		},
		{
			name:               "skip transition before target reached",
			targetPhase:        &applyingState,
			targetPhaseReached: false,
			transition:         structs.RunStateTransition{State: planningState, Terminal: false},
			wantSkip:           true,
			wantTerminal:       false,
			wantError:          false,
		},
		{
			name:               "terminal transition before target reached - error",
			targetPhase:        &applyingState,
			targetPhaseReached: false,
			transition:         structs.RunStateTransition{State: planningState, Terminal: true},
			wantSkip:           false,
			wantTerminal:       false,
			wantError:          true,
			wantErrorMsg:       "filtering failed",
		},
		{
			name:               "terminal state reached without hitting target phase",
			targetPhase:        &applyingState,
			targetPhaseReached: false,
			transition:         structs.RunStateTransition{State: finishedState, Terminal: true},
			wantSkip:           false,
			wantTerminal:       false,
			wantError:          true,
			wantErrorMsg:       "filtering failed",
		},
		{
			name:               "terminal state after target phase was reached",
			targetPhase:        &applyingState,
			targetPhaseReached: true,
			transition:         structs.RunStateTransition{State: finishedState, Terminal: true},
			wantSkip:           true,
			wantTerminal:       true,
			wantError:          false,
		},
		{
			name:               "non-terminal state after target phase reached",
			targetPhase:        &planningState,
			targetPhaseReached: true,
			transition:         structs.RunStateTransition{State: applyingState, Terminal: false},
			wantSkip:           true,
			wantTerminal:       false,
			wantError:          false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			explorer := &Explorer{
				stack:              "test-stack",
				run:                "test-run",
				targetPhase:        tc.targetPhase,
				targetPhaseReached: tc.targetPhaseReached,
			}

			sink := make(chan string, 10)
			defer close(sink)

			skip, terminal, err := explorer.processTargetPhase(&tc.transition, sink)

			if skip != tc.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tc.wantSkip)
			}

			if terminal != tc.wantTerminal {
				t.Errorf("terminal = %v, want %v", terminal, tc.wantTerminal)
			}

			if tc.wantError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if !strings.Contains(err.Error(), tc.wantErrorMsg) {
					t.Errorf("error message = %q, want to contain %q", err.Error(), tc.wantErrorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check that targetPhaseReached is set when we hit the target
			if tc.targetPhase != nil && tc.transition.State == *tc.targetPhase {
				if !explorer.targetPhaseReached {
					t.Errorf("targetPhaseReached should be true when target phase is reached")
				}
			}

			// Check that error messages are sent to sink when appropriate
			if tc.wantError {
				select {
				case msg := <-sink:
					if !strings.Contains(msg, "Run completed without reaching phase") {
						t.Errorf("expected error message in sink, got: %q", msg)
					}
				default:
					t.Errorf("expected error message in sink but sink is empty")
				}
			}
		})
	}
}

func TestProcessTargetPhaseTerminalFlag(t *testing.T) {
	// Test that the terminal flag is correctly passed through when skipping after target reached
	testState := structs.RunState("APPLYING")
	targetState := structs.RunState("PLANNING")

	testCases := []struct {
		name             string
		transitionState  structs.RunState
		terminal         bool
		expectedTerminal bool
	}{
		{
			name:             "terminal transition after target reached returns terminal=true",
			transitionState:  testState,
			terminal:         true,
			expectedTerminal: true,
		},
		{
			name:             "non-terminal transition after target reached returns terminal=false",
			transitionState:  testState,
			terminal:         false,
			expectedTerminal: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			explorer := &Explorer{
				stack:              "test-stack",
				run:                "test-run",
				targetPhase:        &targetState,
				targetPhaseReached: true, // Target already reached
			}

			transition := structs.RunStateTransition{
				State:    tc.transitionState,
				Terminal: tc.terminal,
			}

			sink := make(chan string, 10)
			defer close(sink)

			skip, terminal, err := explorer.processTargetPhase(&transition, sink)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !skip {
				t.Errorf("expected skip=true for transitions after target phase reached")
			}

			if terminal != tc.expectedTerminal {
				t.Errorf("terminal = %v, want %v", terminal, tc.expectedTerminal)
			}
		})
	}
}
