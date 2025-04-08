package version

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// Command returns the CLI version.
func Command(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print out CLI version",
		Action: func(context.Context, *cli.Command) error {
			_, err := fmt.Fprintln(os.Stdout, version)
			return err
		},
	}
}
