package session

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// New creates a session using the default chain of credentials sources:
// first the environment, then the current credentials file.
func New(ctx context.Context, client *http.Client) (Session, error) {
	session, envErr := FromEnvironment(ctx, client)(os.LookupEnv)
	if envErr == nil {
		return session, nil
	}

	session, fileErr := FromCurrentProfile(ctx, client)
	if fileErr == nil {
		return session, nil
	}

	return nil, fmt.Errorf("could not build the session from the environment (%v) or file (%v)", envErr, fileErr)
}
