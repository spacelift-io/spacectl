package workerpools

import "github.com/urfave/cli/v2"

var flagPoolIDNamed = &cli.StringFlag{
	Name:     "pool-id",
	Usage:    "[Required] ID of the worker pool",
	Required: true,
}
