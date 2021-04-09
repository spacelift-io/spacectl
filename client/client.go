package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"

	"github.com/spacelift-io/spacectl/client/session"
)

type client struct {
	wraps   *http.Client
	session session.Session
}

// New returns a new instance of a Spacelift Client.
func New(wraps *http.Client, session session.Session) Client {
	return &client{wraps: wraps, session: session}
}

func (c *client) Mutate(ctx context.Context, mutation interface{}, variables map[string]interface{}) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	return apiClient.Mutate(ctx, mutation, variables)
}

func (c *client) Query(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	return apiClient.Query(ctx, query, variables)
}

func (c *client) URL(format string, a ...interface{}) string {
	endpoint := c.session.Endpoint()

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		panic(err) // Impossible condition.
	}

	endpointURL.Path = fmt.Sprintf(format, a...)

	return endpointURL.String()
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
