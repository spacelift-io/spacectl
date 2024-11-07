package profile

import (
	"errors"
	"fmt"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/urfave/cli/v2"
)

var (
	apiTokenProfile *session.Profile
)

func getAliasWithAPITokenProfile(cliCtx *cli.Context) error {
	ok, err := setGlobalProfileAlias(cliCtx)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	manager, err := session.UserProfileManager()
	if err != nil {
		return fmt.Errorf("could not accesss profile manager: %w", err)
	}

	profile := manager.Current()
	if profile != nil && profile.Credentials.Type == session.CredentialsTypeAPIToken {
		apiTokenProfile = profile
	} else {
		return errors.New("command is only supported when using an existing API Token profile. Please use `spacectl profile login <alias>` instead")
	}

	return nil
}
