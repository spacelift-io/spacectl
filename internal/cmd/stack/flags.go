package stack

import "github.com/urfave/cli/v2"

// flagStackID is flag used for passing the ID for a stack.
//
// It should never be retreived direcly but rather through the getStackID func.
var flagStackID = &cli.StringFlag{
	Name:  "id",
	Usage: "[Optional] User-facing `ID` (slug) of the stack, if not provided stack search is used lookup the stack ID by the current directory and repository name",
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

var flagProjectRootOnly = &cli.BoolFlag{
	Name:  "project-root-only",
	Usage: "Indicate if spacelift should only package files inside this stacks project_root",
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
	Usage: "[Optional] `ID` of the run",
}

var flagRunLatest = &cli.BoolFlag{
	Name:  "run-latest",
	Usage: "[Optional] Indicates that the latest run should be used",
}

var flagNoInit = &cli.BoolFlag{
	Name:  "noinit",
	Usage: "Any pre-initialization hooks as well the vendor-specific initialization procedure will be skipped",
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

var flagAutoConfirm = &cli.BoolFlag{
	Name:     "auto-confirm",
	Usage:    "Indicate whether to automatically confirm the run. It also forces the run log tailing.",
	Value:    false,
	Required: false,
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

var flagOverrideEnvVarsTF = &cli.StringSliceFlag{
	Name:  "tf-env-var-override",
	Usage: "[Optional] Terraform environment variables injected into the run at runtime, they will be prefixed with TF_ by default, example: --tf-env-var-override 'foo=bar,bar=baz'",
}

var flagOverrideEnvVars = &cli.StringSliceFlag{
	Name:  "env-var-override",
	Usage: "[Optional] Environment variables injected into the run at runtime, example: --env-var-override 'foo=bar,bar=baz'",
}

var flagDisregardGitignore = &cli.BoolFlag{
	Name:  "disregard-gitignore",
	Usage: "[Optional] Disregard the .gitignore file when reading files in a directory",
}

var flagResources = &cli.StringSliceFlag{
	Name:  "resources",
	Usage: "[Optional] A comma separeted list of resources to be used when applying, example: 'aws_instance.foo'",
}

var flagPrioritizeRun = &cli.BoolFlag{
	Name:  "prioritize-run",
	Usage: "[Optional] Indicate whether to prioritize the run",
}

var flagInteractive = &cli.BoolFlag{
	Name:    "interactive",
	Aliases: []string{"i"},
	Usage:   "[Optional] Whether to run the command in interactive mode",
}
