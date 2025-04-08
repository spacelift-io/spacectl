package blueprint

import "github.com/urfave/cli/v3"

var flagRequiredBlueprintID = &cli.StringFlag{
	Name:     "blueprint-id",
	Aliases:  []string{"b-id"},
	Usage:    "[Required] `ID` of the blueprint",
	Required: true,
}
