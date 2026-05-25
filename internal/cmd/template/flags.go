package template

import "github.com/urfave/cli/v3"

var flagRequiredTemplateID = &cli.StringFlag{
	Name:     "template-id",
	Aliases:  []string{"t-id"},
	Usage:    "[Required] `ID` of the template",
	Required: true,
}
