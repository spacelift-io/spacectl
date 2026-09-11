package session

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// New creates a session using the default chain of credentials sources:
// first the environment, then the current credentials file.
//
// Naming a profile through SPACELIFT_PROFILE is an explicit choice, so it wins over credentials
// that merely happen to be present in the environment: those are skipped entirely, rather than
// risk authenticating as somebody other than the profile that was asked for.
func New(ctx context.Context, client *http.Client) (Session, error) {
	skipEnvironment := os.Getenv(EnvSpaceliftProfile) != ""

	var envErr error
	if !skipEnvironment {
		session, err := FromEnvironment(ctx, client)(os.LookupEnv)
		if err == nil {
			return session, nil
		}

		envErr = err
	}

	session, fileErr := FromCurrentProfile(ctx, client)
	if fileErr == nil {
		return session, nil
	}

	if skipEnvironment {
		return nil, fileErr
	}

	return nil, fmt.Errorf("could not build the session from the environment (%v) or file (%v)", envErr, fileErr)
}
