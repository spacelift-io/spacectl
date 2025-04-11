package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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

func (c *client) Mutate(ctx context.Context, mutation interface{}, variables map[string]interface{}, opts ...graphql.RequestOption) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	err = apiClient.Mutate(ctx, mutation, variables, opts...)

	if err != nil && err.Error() == "unauthorized" {
		return fmt.Errorf("unauthorized: you can re-login using `spacectl profile login`")
	}

	return err
}

func (c *client) Query(ctx context.Context, query interface{}, variables map[string]interface{}, opts ...graphql.RequestOption) error {
	apiClient, err := c.apiClient(ctx)
	if err != nil {
		return nil
	}

	err = apiClient.Query(ctx, query, variables, opts...)

	if err != nil && err.Error() == "unauthorized" {
		return fmt.Errorf("unauthorized: you can re-login using `spacectl profile login`")
	}

	return err
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
	httpC, err := c.httpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("graphql client creation failed at http client creation: %w", err)
	}

	requestOptions := []graphql.RequestOption{
		graphql.WithHeader("Spacelift-Client-Type", "spacectl"),
	}

	return graphql.NewClient(c.session.Endpoint(), httpC, requestOptions...), nil
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	// get http client
	httpC, err := c.httpClient(req.Context())
	if err != nil {
		return nil, fmt.Errorf("http client creation failed: %w", err)
	}

	// prepend request URL with spacelift endpoint
	endpoint := strings.TrimRight(c.session.Endpoint(), "/graphql")
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host

	// execute request
	resp, err := httpC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: you can re-login using `spacectl profile login`")
	}
	return resp, err
}

func (c *client) httpClient(ctx context.Context) (*http.Client, error) {
	bearerToken, err := c.session.BearerToken(ctx)
	if err != nil {
		return nil, err
	}

	return oauth2.NewClient(
		context.WithValue(ctx, oauth2.HTTPClient, c.wraps), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: bearerToken},
		),
	), nil
}
