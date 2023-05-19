package run_external_dependency

import "github.com/urfave/cli/v2"

var flagRunExternalDependencyID = &cli.StringFlag{
	Name:     "id",
	Usage:    "[Required] ID of the external dependency",
	Required: true,
}

var flagStatus = &cli.StringFlag{
	Name:     "status",
	Usage:    "[Required] Status of the external dependency (one of: 'finished', 'failed')",
	Required: true,
}
