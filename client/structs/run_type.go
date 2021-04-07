package structs

// RunType is the type of the run.
type RunType string

func NewRunType(in string) *RunType {
	out := RunType(in)
	return &out
}
