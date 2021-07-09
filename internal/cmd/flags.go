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
