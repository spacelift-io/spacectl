package module

import "github.com/urfave/cli/v2"

var flagModuleID = &cli.StringFlag{
	Name:     "id",
	Usage:    "[Required] User-facing `ID` (slug) of the module",
	Required: true,
}

var flagCommitSHA = &cli.StringFlag{
	Name:  "sha",
	Usage: "Commit `SHA` to use for the module version",
}

var flagNoFindRepositoryRoot = &cli.BoolFlag{
	Name:  "no-find-repository-root",
	Usage: "Indicate whether spacectl should avoid finding the repository root (containing a .git directory) before packaging it.",
	Value: false,
}

var flagNoUpload = &cli.BoolFlag{
	Name:  "no-upload",
	Usage: "Indicate whether Spacectl should prepare the workspace archive, but skip uploading it. Useful for debugging ignorefiles.",
	Value: false,
}

var flagRunMetadata = &cli.StringFlag{
	Name:  "run-metadata",
	Usage: "Additional opaque metadata you will be able to access from policies handling this Run.",
}

var flagVersion = &cli.StringFlag{
	Name: "version",
	Usage: "Semver `version` for the module version. If not provided, the version " +
		"from the configuration file will be used",
}

var flagDisregardGitignore = &cli.BoolFlag{
	Name:  "disregard-gitignore",
	Usage: "[Optional] Disregard the .gitignore file when reading files in a directory",
}

var flagTests = &cli.StringSliceFlag{
	Name:  "tests",
	Usage: "[Optional] A list of test IDs to run",
}
