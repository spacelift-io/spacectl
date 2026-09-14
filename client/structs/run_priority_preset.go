package structs

// RunPriorityPreset is the scheduling tier a run occupies in its worker pool queue.
type RunPriorityPreset string

// NewRunPriorityPreset returns a pointer suitable for the runPrioritySet mutation's preset argument.
func NewRunPriorityPreset(in string) *RunPriorityPreset {
	out := RunPriorityPreset(in)
	return &out
}
