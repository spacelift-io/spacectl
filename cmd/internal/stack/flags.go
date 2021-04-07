package stack

import "github.com/urfave/cli/v2"

var flagStackID = &cli.StringFlag{
	Name:     "id",
	Usage:    "User-facing ID (slug) of the stack",
	Required: true,
}

var flagCommitSHA = &cli.StringFlag{
	Name:  "sha",
	Usage: "Commit SHA for the newly created run",
}

var flagRun = &cli.StringFlag{
	Name:     "run",
	Usage:    "ID of the run",
	Required: true,
}

var flagNoInit = &cli.BoolFlag{
	Name:  "noinit",
	Usage: "Indicate whether to skip initialization for a task",
	Value: false,
}

var flagTail = &cli.BoolFlag{
	Name:  "tail",
	Usage: "Indicate whether to tail the run",
	Value: false,
}
