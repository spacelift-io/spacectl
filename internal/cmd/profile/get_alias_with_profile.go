package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/urfave/cli/v2"
)

var (
	apiTokenProfile *session.Profile
)

func getAliasWithAPITokenProfile(cliCtx *cli.Context) error {
	if err := getAlias(cliCtx); err == nil {
		return nil
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find user home directory: %w", err)
	}

	manager, err := session.NewProfileManager(filepath.Join(userHomeDir, session.SpaceliftConfigDirectory))
	if err != nil {
		return fmt.Errorf("could not create profile manager: %w", err)
	}

	profile := manager.Current()
	if profile != nil && profile.Credentials.Type == session.CredentialsTypeAPIToken {
		apiTokenProfile = profile
	} else {
		return errors.New("command is supported only for exisiting API Token profile. Please use `spacectl profile login <alias>` instead")
	}

	return nil
}
