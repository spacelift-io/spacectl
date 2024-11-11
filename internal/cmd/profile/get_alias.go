package profile

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var profileAlias string

// setGlobalProfileAlias sets the global profile alias if it is provided as the only argument.
// It returns false if no arguments were provided and error if there were more than one.
//
// If false is returned, the caller should attempt to get the profile alias on its own.
func setGlobalProfileAlias(cliCtx *cli.Context) (bool, error) {
	switch cliCtx.NArg() {
	case 0:
		return false, nil
	case 1:
		profileAlias = cliCtx.Args().Get(0)
		return true, nil
	default:
		return false, fmt.Errorf("expecting profile alias as the only argument, got %d instead", cliCtx.NArg())
	}
}
