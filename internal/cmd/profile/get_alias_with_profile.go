package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
)

var (
	apiTokenProfile *session.Profile
)

func getAliasWithAPITokenProfile(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
	ok, err := setGlobalProfileAlias(cliCmd)
	if err != nil {
		return ctx, err
	}

	if ok {
		return ctx, nil
	}

	manager, err := session.UserProfileManager()
	if err != nil {
		return ctx, fmt.Errorf("could not accesss profile manager: %w", err)
	}

	profile := manager.Current()
	if profile != nil && profile.Credentials.Type == session.CredentialsTypeAPIToken {
		apiTokenProfile = profile
	} else {
		return ctx, errors.New("command is only supported when using an existing API Token profile. Please use `spacectl profile login <alias>` instead")
	}

	return ctx, nil
}
