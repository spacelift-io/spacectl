package session

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/shurcooL/graphql"
)

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
}

func (g *apiKey) BearerToken(ctx context.Context) (string, error) {
	if !g.isFresh() {
		if err := g.exchange(ctx); err != nil {
			return "", err
		}
	}

	return g.apiToken.BearerToken(ctx)
}

func (g *apiKey) exchange(ctx context.Context) error {
	var mutation struct {
		APIKeyUser user `graphql:"apiKeyUser(id: $id, secret: $secret)"`
	}

	variables := map[string]interface{}{
		"id":     graphql.ID(g.keyID),
		"secret": graphql.String(g.keySecret),
	}

	if err := g.mutate(ctx, &mutation, variables); err != nil {
		return fmt.Errorf("could not exchange API key and secret for token: %w", err)
	}

	g.setJWT(&mutation.APIKeyUser)

	return nil
}
