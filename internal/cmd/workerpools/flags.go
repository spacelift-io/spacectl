package workerpools

import "github.com/urfave/cli/v3"

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
