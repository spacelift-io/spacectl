package client

import (
	"github.com/hasura/go-graphql-client"
)

// RequestOption allows you to modify the request before sending it to the
// server.
type RequestOption func(*requestOptions)

// WithHeader sets a header on the request.
func WithHeader(key, value string) RequestOption {
	return func(o *requestOptions) {
		o.addHeader(key, value)
	}
}

// WithGraphqlOptions sets [graphql.Option] options.
func WithGraphqlOptions(options ...graphql.Option) RequestOption {
	return func(o *requestOptions) {
		o.graphqlOptions = append(o.graphqlOptions, options...)
	}
}

func parseOptions(options ...RequestOption) requestOptions {
	var opts requestOptions
	for _, opt := range options {
		opt(&opts)
	}
	return opts
}

type requestOptions struct {
	headers        map[string]string
	graphqlOptions []graphql.Option
}

func (o *requestOptions) addHeader(key, value string) {
	if o.headers == nil {
		o.headers = map[string]string{}
	}
	o.headers[key] = value
}
