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

	// aliasFromArgs records whether the alias came from the command line rather than from
	// the profile in use. Persisting the login depends on the difference: an alias the user
	// typed is a deliberate choice, so it becomes the stored selection even when
	// SPACELIFT_PROFILE names the same profile.
	aliasFromArgs bool
)

func getAliasWithAPITokenProfile(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
	ok, err := setGlobalProfileAlias(cliCmd)
	if err != nil {
		return ctx, err
	}

	aliasFromArgs = ok

	if ok {
		return ctx, nil
	}

	manager, err := session.UserProfileManager()
	if err != nil {
		return ctx, fmt.Errorf("could not accesss profile manager: %w", err)
	}

	// Only this branch depends on the override resolving, so validate here rather than for
	// the whole command: `SPACELIFT_PROFILE=new spacectl profile login new` must stay able
	// to create the profile the variable names.
	profile, err := manager.CurrentValidated()
	if err != nil {
		return ctx, err
	}
	if profile != nil && profile.Credentials.Type == session.CredentialsTypeAPIToken {
		apiTokenProfile = profile
	} else {
		return ctx, errors.New("command is only supported when using an existing API Token profile. Please use `spacectl profile login <alias>` instead")
	}

	return ctx, nil
}
