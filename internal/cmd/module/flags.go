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

var flagVersion = &cli.StringFlag{
	Name: "version",
	Usage: "Semver `version` for the module version. If not provided, the version " +
		"from the configuration file will be used",
}
