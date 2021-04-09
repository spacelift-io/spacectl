package stack

import (
	"github.com/urfave/cli/v2"
)

var stackID string

func beforeEach(cliCtx *cli.Context) error {
	stackID = cliCtx.String(flagStackID.Name)
	return nil
}
