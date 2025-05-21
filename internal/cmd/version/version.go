package version

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
)

// Command returns the CLI version.
func Command(spacectlVersion string, spaceliftVersion cmd.SpaceliftInstanceVersion) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print out CLI version",
		Action: func(_ context.Context, _ *cli.Command) error {
			_, err := fmt.Fprintf(os.Stdout, "spacectl version: %s, Spacelift version: %s\n", spacectlVersion, spaceliftVersion.String())
			return err
		},
	}
}
