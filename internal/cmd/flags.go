package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

// FlagOutputFormat allows users to change the output format of commands that support it.
var FlagOutputFormat = &cli.StringFlag{
	Name:    "output",
	Aliases: []string{"o"},
	Usage:   fmt.Sprintf("Output `format`. Allowed values: %s", strings.Join(AvailableOutputFormatStrings, ", ")),
	Value:   string(OutputFormatTable),
}

// FlagNoColor disables coloring in the console output.
var FlagNoColor = &cli.BoolFlag{
	Name:  "no-color",
	Usage: "Disables coloring for the console output. Automatically enabled when the output is not a terminal.",
}
