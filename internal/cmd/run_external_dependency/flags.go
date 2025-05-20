package runexternaldependency

import "github.com/urfave/cli/v3"

var flagRunID = &cli.StringFlag{
	Name:     "run-id",
	Usage:    "[Required] ID of the run",
	Required: true,
}

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
