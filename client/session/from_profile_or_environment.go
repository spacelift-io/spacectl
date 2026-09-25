package session

import (
	"context"
	"net/http"
)

// FromProfileOrEnvironment returns the session for commands that hand a token to
// another tool. It prefers the currently selected profile, falling back to
// credentials from the environment when none is set (e.g. in CI, where
// SPACELIFT_API_KEY_ID and SPACELIFT_API_KEY_SECRET - the latter possibly an OIDC
// token - are provided directly). The fallback lets those commands exchange the
// credentials for a session token without anyone having to hand-roll the
// apiKeyUser GraphQL mutation.
func FromProfileOrEnvironment(ctx context.Context, m *ProfileManager, httpClient *http.Client, lookup func(string) (string, bool)) (Session, error) {
	if m != nil {
		// Naming a profile through SPACELIFT_PROFILE is an explicit choice, so a bad alias is
		// an error rather than a reason to fall back and print a token for a different account.
		current, err := m.CurrentValidated()
		if err != nil {
			return nil, err
		}

		if current != nil {
			return current.Credentials.Session(ctx, httpClient)
		}
	}

	return FromEnvironment(ctx, httpClient)(lookup)
}
