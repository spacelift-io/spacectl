package client

import (
	"context"

	"github.com/shurcooL/graphql"
)

// Client abstracts away Spacelift's client API.
type Client interface {
	// Query executes a single GraphQL query request.
	Query(context.Context, interface{}, map[string]interface{}, ...graphql.RequestOption) error

	// Mutate executes a single GraphQL mutation request.
	Mutate(context.Context, interface{}, map[string]interface{}, ...graphql.RequestOption) error

	// URL returns a full URL given a formatted path.
	URL(string, ...interface{}) string
}
