package client

import (
	"net/http"

	"github.com/hasura/go-graphql-client"
)

type RequestOption func(*requestOptions)

// WithHeader sets a header on the request.
func WithHeader(key, value string) RequestOption {
	return func(o *requestOptions) {
		o.modifyRequest = append(o.modifyRequest, func(request *http.Request) {
			request.Header.Set(key, value)
		})
	}
}

// WithModifyRequest allows you to modify the request before sending it to the server.
func WithModifyRequest(f func(request *http.Request)) RequestOption {
	return func(o *requestOptions) {
		o.modifyRequest = append(o.modifyRequest, f)
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
	modifyRequest  []func(*http.Request)
	graphqlOptions []graphql.Option
}
