package session

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// FromProfileOrEnvironment resolves a session for commands that pass a token to
// another tool. It prefers the selected profile and uses environment credentials
// when no profile is selected, so CI jobs using API keys or OIDC can get a token
// without implementing the exchange themselves. It rejects expired stored tokens
// before the receiving tool tries to use them.
func FromProfileOrEnvironment(ctx context.Context, m *ProfileManager, httpClient *http.Client, lookup func(string) (string, bool)) (Session, error) {
	var (
		sess Session
		err  error
	)

	switch current, profileErr := currentProfile(m); {
	case profileErr != nil:
		return nil, profileErr
	case current != nil:
		sess, err = current.Credentials.Session(ctx, httpClient)
	default:
		sess, err = FromEnvironment(ctx, httpClient)(lookup)
	}
	if err != nil {
		return nil, err
	}

	// Stored tokens aren't refreshed. Reject expired ones here so the user gets
	// login advice instead of an authentication failure in the receiving tool.
	if token, ok := sess.(*apiToken); ok && !token.tokenValidUntil.IsZero() && !token.timer().Before(token.tokenValidUntil) {
		return nil, fmt.Errorf("the Spacelift token expired at %s; log in again with `spacectl profile login`, or set fresh credentials in the environment",
			token.tokenValidUntil.Format(time.RFC3339))
	}

	return sess, nil
}

func currentProfile(m *ProfileManager) (*Profile, error) {
	if m == nil {
		return nil, nil
	}

	// Falling back after an invalid SPACELIFT_PROFILE could return a token for
	// a different account than the user requested.
	return m.CurrentValidated()
}
