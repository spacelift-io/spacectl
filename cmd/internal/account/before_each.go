package account

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/client/session"
)

var (
	accountAlias string
	aliasPath    string
	configDir    string
	currentPath  string
)

func beforeEach(cliCtx *cli.Context) error {
	if nArgs := cliCtx.NArg(); nArgs != 2 {
		return fmt.Errorf("expecting account alias as the only argument, got %d instead", nArgs)
	}

	accountAlias = cliCtx.Args().Get(1)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	configDir = filepath.Join(homeDir, session.SpaceliftConfigDirectory)
	aliasPath = filepath.Join(configDir, accountAlias)
	currentPath = filepath.Join(configDir, session.CurrentFileName)

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("could not create Spacelift config directory: %w", err)
	}

	return nil
}
