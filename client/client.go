package client

import (
	"context"
	"net/http"

	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"

	"github.com/spacelift-io/spacelift-cli/client/session"
)

type client struct {
	wraps   *http.Client
	session session.Session
}

func (c *client) Query(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	return apiClient.Query(ctx, query, variables)
}

func (c *client) Mutate(ctx context.Context, mutation interface{}, variables map[string]interface{}) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	return apiClient.Mutate(ctx, mutation, variables)
}

func (c *client) apiClient(ctx context.Context) (*graphql.Client, error) {
	bearerToken, err := c.session.BearerToken(ctx)
	if err != nil {
		return nil, err
	}

	return graphql.NewClient(c.session.Endpoint(), oauth2.NewClient(
		context.WithValue(ctx, oauth2.HTTPClient, c.wraps), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: bearerToken},
		),
	)), nil
}
