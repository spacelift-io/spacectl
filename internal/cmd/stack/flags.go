package stack

import "github.com/urfave/cli/v2"

// flagStackID is flag used for passing the ID for a stack.
//
// It should never be retreived direcly but rather through the getStackID func.
var flagStackID = &cli.StringFlag{
	Name:  "id",
	Usage: "[Optional] User-facing `ID` (slug) of the stack, if not provided environment variable SPACECTL_STACK_ID is used or spacectl tries to lookup the stack by the current directory and repository name",
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

var flagRequiredRun = &cli.StringFlag{
	Name:     "run",
	Usage:    "[Required] `ID` of the run",
	Required: true,
}

var flagRun = &cli.StringFlag{
	Name:  "run",
	Usage: "`ID` of the run",
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

var flagNoUpload = &cli.BoolFlag{
	Name:  "no-upload",
	Usage: "Indicate whether Spacectl should prepare the workspace archive, but skip uploading it. Useful for debugging ignorefiles.",
	Value: false,
}

var flagIgnoreSubdir = &cli.BoolFlag{
	Name:  "ignore-subdir",
	Usage: "[Optional] Indicate whetever open command should ignore the current subdir and only use the repository to search for a stack",
	Value: false,
}

var flagCurrentBranch = &cli.BoolFlag{
	Name:  "current-branch",
	Usage: "[Optional] Indicate whetever to search a stack by the current branch you're on",
	Value: false,
}

var flagSearchCount = &cli.IntFlag{
	Name:  "count",
	Usage: "[Optional] Indicate the maximum count of elements returned when searching stacks",
	Value: 30,
}

var flagShowLabels = &cli.BoolFlag{
	Name:     "show-labels",
	Usage:    "[Optional] Indicates that stack labels should be printed when outputting stack data in the table format",
	Required: false,
	Value:    false,
}
