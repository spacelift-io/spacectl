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
		return fromAPIKeySecretSource(ctx, client, endpoint, keyID, func(context.Context) (string, error) {
			return keySecret, nil
		})
	}
}

// fromAPIKeySecretSource builds an API key session that asks keySecret for the
// secret on every exchange, so a short-lived secret like a CI OIDC token can be
// minted again when the session goes stale.
func fromAPIKeySecretSource(ctx context.Context, client *http.Client, endpoint, keyID string, keySecret func(context.Context) (string, error)) (Session, error) {
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

type apiKey struct {
	apiToken
	keyID     string
	keySecret func(context.Context) (string, error)
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

	secret, err := g.keySecret(ctx)
	if err != nil {
		return err
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
