package blueprint

import "github.com/urfave/cli/v2"

var flagShowLabels = &cli.BoolFlag{
	Name:     "show-labels",
	Usage:    "[Optional] Indicates that stack labels should be printed when outputting stack data in the table format",
	Required: false,
	Value:    false,
}

var flagLimit = &cli.UintFlag{
	Name:     "limit",
	Usage:    "[Optional] Limit the number of items to return",
	Required: false,
}

var flagSearch = &cli.StringFlag{
	Name:     "search",
	Usage:    "[Optional] Performs a full-text search.",
	Required: false,
}
