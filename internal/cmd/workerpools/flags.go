package workerpools

import "github.com/urfave/cli/v3"

var flagWorkerPoolIDOptional = &cli.StringFlag{
	Name:  "pool-id",
	Usage: "[Optional] ID of the worker pool; omit to use the shared public worker pool",
}

var flagPoolIDNamed = &cli.StringFlag{
	Name:     "pool-id",
	Usage:    "[Required] ID of the worker pool",
	Required: true,
}

var flagWorkerID = &cli.StringFlag{
	Name:     "id",
	Usage:    "[Required] ID of the worker",
	Required: true,
}

var flagWaitUntilDrained = &cli.BoolFlag{
	Name:     "wait-until-drained",
	Usage:    "Wait until the worker is drained",
	Required: false,
}

var flagQueueLimit = &cli.IntFlag{
	Name:  "limit",
	Usage: "Maximum number of schedulable runs to fetch",
	Value: 100,
}

var flagQueueAfter = &cli.StringFlag{
	Name:  "after",
	Usage: "Pagination cursor from a previous queue list (JSON output includes pageInfo.endCursor)",
}

var flagQueueSearch = &cli.StringFlag{
	Name:  "search",
	Usage: "Full-text search across schedulable runs",
}

var flagQueueState = &cli.StringSliceFlag{
	Name:    "state",
	Usage:   "Only include runs in this state (repeatable), e.g. QUEUED",
	Sources: cli.EnvVars("SPACECTL_QUEUE_STATE"),
}

var flagQueueExcludeState = &cli.StringSliceFlag{
	Name:    "exclude-state",
	Usage:   "Exclude runs in this state (repeatable)",
	Sources: cli.EnvVars("SPACECTL_QUEUE_EXCLUDE_STATE"),
}

var flagQueueStack = &cli.StringFlag{
	Name:  "stack",
	Usage: "Only include runs for this stack id (slug); matched client-side after listing from the API",
}

var flagQueueRunType = &cli.StringSliceFlag{
	Name:    "run-type",
	Usage:   "Only include runs of this type (repeatable); values are uppercased for the API (e.g. tracked and TRACKED both work)",
	Sources: cli.EnvVars("SPACECTL_QUEUE_RUN_TYPE"),
}

var flagQueueDriftDetection = &cli.StringFlag{
	Name:  "drift-detection",
	Usage: "Only include runs with this drift detection setting (true or false); matched client-side (not a search predicate)",
}

var flagQueuePrioritized = &cli.StringFlag{
	Name:  "prioritized",
	Usage: "Only include runs where isPrioritized is true or false (matches API Run.isPrioritized)",
}

var flagQueueCommit = &cli.StringFlag{
	Name:  "commit",
	Usage: "Only include runs whose commit hash contains this substring (case-insensitive, matched client-side)",
}

var flagQueueRun = &cli.StringFlag{
	Name:  "run",
	Usage: "When discarding a single run, its id (requires --stack)",
}

var flagQueueYes = &cli.BoolFlag{
	Name:  "yes",
	Usage: "Required to confirm filter-based discard of multiple runs",
	Value: false,
}
