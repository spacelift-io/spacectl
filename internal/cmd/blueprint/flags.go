package blueprint

import "github.com/urfave/cli/v3"

var flagRequiredBlueprintID = &cli.StringFlag{
	Name:     "blueprint-id",
	Aliases:  []string{"b-id"},
	Usage:    "[Required] `ID` of the blueprint",
	Required: true,
}

var flagInputFile = &cli.StringFlag{
	Name:    "input-file",
	Aliases: []string{"if"},
	Usage:   "[Optional] Load blueprint options from the JSON `FILE`",
}
