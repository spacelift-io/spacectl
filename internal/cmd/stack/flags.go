package stack

import "github.com/urfave/cli/v2"

var flagStackID = &cli.StringFlag{
	Name:     "id",
	Usage:    "[Required] User-facing `ID` (slug) of the stack",
	Required: true,
}

var flagCommitSHA = &cli.StringFlag{
	Name:  "sha",
	Usage: "Commit `SHA` for the newly created run",
}

var flagOutputID = &cli.StringFlag{
	Name:  "output-id",
	Usage: "`ID` of output",
}

var flagEnvironmentWriteOnly = &cli.BoolFlag{
	Name:  "write-only",
	Usage: "Indicates whether the content can be read back outside a Run",
	Value: false,
}

var flagNoFindRepositoryRoot = &cli.BoolFlag{
	Name:  "no-find-repository-root",
	Usage: "Indicate whether spacectl should avoid finding the repository root (containing a .git directory) before packaging it.",
	Value: false,
}

var flagRequiredCommitSHA = &cli.StringFlag{
	Name:     "sha",
	Usage:    "[Required] `SHA` of the commit to set as canonical for the stack",
	Required: true,
}

var flagRun = &cli.StringFlag{
	Name:     "run",
	Usage:    "[Required] `ID` of the run",
	Required: true,
}

var flagNoInit = &cli.BoolFlag{
	Name:  "noinit",
	Usage: "Indicate whether to skip initialization for a task",
	Value: false,
}

var flagRunMetadata = &cli.StringFlag{
	Name:  "run-metadata",
	Usage: "Additional opaque metadata you will be able to access from policies handling this Run.",
}

var flagTail = &cli.BoolFlag{
	Name:  "tail",
	Usage: "Indicate whether to tail the run",
	Value: false,
}

var flagNoTail = &cli.BoolFlag{
	Name:  "no-tail",
	Usage: "Indicate whether not to tail the run",
	Value: false,
}

var flagMaxResults = &cli.IntFlag{
	Name:  "max-results",
	Usage: "The maximum number of items to return",
	Value: 10,
}
