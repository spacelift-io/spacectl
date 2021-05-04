package version

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// Command returns the CLI version.
func Command(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print out CLI version",
		Action: func(*cli.Context) error {
			_, err := fmt.Fprintln(os.Stdout, version)
			return err
		},
	}
}
