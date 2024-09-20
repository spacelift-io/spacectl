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

// FlagLimit is a flag used for limiting the number of items to return.
var FlagLimit = &cli.UintFlag{
	Name:  "limit",
	Usage: "[Optional] Limit the number of items to return",
}

// FlagSearch is a flag used for performing a full-text search.
var FlagSearch = &cli.StringFlag{
	Name:  "search",
	Usage: "[Optional] Performs a full-text search.",
}

// FlagShowLabels is a flag used for indicating that labels should be printed when outputting data in the table format.
var FlagShowLabels = &cli.BoolFlag{
	Name:     "show-labels",
	Usage:    "[Optional] Indicates that labels should be printed when outputting data in the table format",
	Required: false,
	Value:    false,
}
