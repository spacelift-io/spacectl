package session

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/shurcooL/graphql"
)

// FromAPIKey builds a Spacelift session from a combination of endpoint, API key
// ID and API key secret.
func FromAPIKey(ctx context.Context, client *http.Client) func(string, string, string) (Session, error) {
	return func(endpoint, keyID, keySecret string) (Session, error) {
		out := &apiKey{
			apiToken: apiToken{
				client:   client,
				endpoint: endpoint,
				timer:    time.Now,
			},
			keyID:     keyID,
			keySecret: keySecret,
		}

		if err := out.exchange(ctx); err != nil {
			return nil, err
		}

		return out, nil
	}
}

type apiKey struct {
	apiToken
	keyID, keySecret string

	// githubOIDC, when set, mints the secret for every exchange in place of keySecret, since
	// GitHub's OIDC tokens expire long before the Spacelift session does.
	githubOIDC *githubOIDC
}

func (g *apiKey) BearerToken(ctx context.Context) (string, error) {
	if !g.isFresh() {
		if err := g.exchange(ctx); err != nil {
			return "", err
		}
	}

	return g.apiToken.BearerToken(ctx)
}

func (g *apiKey) Type() CredentialsType {
	return CredentialsTypeAPIKey
}

func (g *apiKey) exchange(ctx context.Context) error {
	var mutation struct {
		APIKeyUser user `graphql:"apiKeyUser(id: $id, secret: $secret)"`
	}

	secret := g.keySecret
	if g.githubOIDC != nil {
		var err error
		if secret, err = g.githubOIDC.mint(ctx); err != nil {
			return err
		}
	}

	variables := map[string]any{
		"id":     graphql.ID(g.keyID),
		"secret": graphql.String(secret),
	}

	if err := g.mutate(ctx, &mutation, variables); err != nil {
		return fmt.Errorf("could not exchange API key and secret for token: %w", err)
	}

	g.setJWT(&mutation.APIKeyUser)

	return nil
}
