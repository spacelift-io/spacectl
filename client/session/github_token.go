package session

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// FromGitHubToken builds a Spacelift session from a combination of endpoint,
// and a GitHub access token.
func FromGitHubToken(ctx context.Context, client *http.Client) func(string, string) (Session, error) {
	return func(endpoint, accessToken string) (Session, error) {
		out := &gitHubToken{
			apiToken: apiToken{
				client:   client,
				endpoint: endpoint,
				timer:    time.Now,
			},
			accessToken: accessToken,
		}

		if err := out.exchange(ctx); err != nil {
			return nil, err
		}

		return out, nil
	}
}

type gitHubToken struct {
	apiToken
	accessToken string
}

func (g *gitHubToken) BearerToken(ctx context.Context) (string, error) {
	if !g.isFresh() {
		if err := g.exchange(ctx); err != nil {
			return "", err
		}
	}

	return g.apiToken.BearerToken(ctx)
}

func (g *gitHubToken) Type() CredentialsType {
	return CredentialsTypeGitHubToken
}

func (g *gitHubToken) exchange(ctx context.Context) error {
	var mutation struct {
		APIKeyUser user `graphql:"oauthUser(token: $token)"`
	}

	variables := map[string]interface{}{"token": g.accessToken}

	if err := g.mutate(ctx, &mutation, variables); err != nil {
		return fmt.Errorf("could not exchange access token for Spacelift one: %w", err)
	}

	g.setJWT(&mutation.APIKeyUser)

	return nil
}
