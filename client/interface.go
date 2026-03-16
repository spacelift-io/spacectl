package client

import (
	"context"
	"net/http"

	"github.com/shurcooL/graphql"
)

// Client abstracts away Spacelift's client API.
type Client interface {
	// Query executes a single GraphQL query request.
	Query(context.Context, any, map[string]any, ...graphql.RequestOption) error

	// Mutate executes a single GraphQL mutation request.
	Mutate(context.Context, any, map[string]any, ...graphql.RequestOption) error

	// URL returns a full URL given a formatted path.
	URL(string, ...any) string

	// Do executes an authenticated http request to the Spacelift API
	Do(r *http.Request) (*http.Response, error)
}
