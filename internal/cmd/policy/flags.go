package policy

import "github.com/urfave/cli/v2"

var flagRequiredPolicyID = &cli.StringFlag{
	Name:     "id",
	Usage:    "[Required] `ID` of the policy",
	Required: true,
}

var flagRequiredSampleKey = &cli.StringFlag{
	Name:     "key",
	Usage:    "[Required] `Key` of the policy sample",
	Required: true,
}

var flagSimulationInput = &cli.StringFlag{
	Name:     "input",
	Usage:    "[Required] JSON Input of the data provided for policy simlation. Will Attempt to detect if a file is provided",
	Required: true,
}
