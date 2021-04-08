package profile

import (
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

var (
	profileAlias string
	aliasPath    string
)

func getAlias(cliCtx *cli.Context) error {
	if nArgs := cliCtx.NArg(); nArgs != 1 {
		return fmt.Errorf("expecting profile alias as the only argument, got %d instead", nArgs)
	}

	profileAlias = cliCtx.Args().Get(0)
	aliasPath = filepath.Join(configDir, profileAlias)

	return nil
}
