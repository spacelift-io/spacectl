package structs

// RunType is the type of the run.
type RunType string

// NewRunType takes a string and returns a pointer to a RunType.
func NewRunType(in string) *RunType {
	out := RunType(in)
	return &out
}
