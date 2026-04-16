package structs

// ForceApplyMode selects how a force-applied tracked run propagates to dependencies.
type ForceApplyMode string

// NewForceApplyMode returns a pointer suitable for the runTrigger mutation's forceApply argument.
func NewForceApplyMode(in string) *ForceApplyMode {
	out := ForceApplyMode(in)
	return &out
}
