package structs

import (
	"fmt"
	"strings"
	"time"
)

// RunStateTransition represents a single run state transition.
type RunStateTransition struct {
	HasLogs   bool     `graphql:"hasLogs"`
	Note      *string  `graphql:"note"`
	State     RunState `graphql:"state"`
	Terminal  bool     `graphql:"terminal"`
	Timestamp int      `graphql:"timestamp"`
	Username  *string  `graphql:"username"`
}

// About returns "header" information about the state transition.
func (r *RunStateTransition) About() string {
	parts := []string{
		string(r.State),
		time.Unix(int64(r.Timestamp), 0).Format(time.UnixDate),
	}

	if username := r.Username; username != nil {
		parts = append(parts, *username)
	}

	if note := r.Note; note != nil {
		parts = append(parts, *note)
	}

	return strings.Join(parts, "\t")
}

func (r *RunStateTransition) Error() error {
	if r.State == RunState("FINISHED") {
		return nil
	}

	return fmt.Errorf("finished with %s state", r.State)
}
