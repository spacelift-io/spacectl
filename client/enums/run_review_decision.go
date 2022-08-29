package enums

// RunReviewDecision represents the API RunReviewDecision enum.
type RunReviewDecision string

const (
	// RunReviewDecisionApprove represents an approval decision.
	RunReviewDecisionApprove = "APPROVE"

	// RunReviewDecisionReject represents a rejection decision.
	RunReviewDecisionReject = "REJECT"
)
