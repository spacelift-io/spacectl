package stack

// RuntimeConfigInput represents the input for triggering a run with a runtime configuration.
type RuntimeConfigInput struct {
	Yaml          *string        `json:"yaml"`
	ProjectRoot   *string        `json:"projectRoot"`
	RunnerImage   *string        `json:"runnerImage"`
	Environment   *[]EnvVarInput `json:"environment"`
	AfterApply    *[]string      `json:"afterApply"`
	AfterDestroy  *[]string      `json:"afterDestroy"`
	AfterInit     *[]string      `json:"afterInit"`
	AfterPerform  *[]string      `json:"afterPerform"`
	AfterPlan     *[]string      `json:"afterPlan"`
	AfterRun      *[]string      `json:"afterRun"`
	BeforeApply   *[]string      `json:"beforeApply"`
	BeforeDestroy *[]string      `json:"beforeDestroy"`
	BeforeInit    *[]string      `json:"beforeInit"`
	BeforePerform *[]string      `json:"beforePerform"`
	BeforePlan    *[]string      `json:"beforePlan"`
}

type EnvVarInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
