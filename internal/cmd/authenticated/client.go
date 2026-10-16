package authenticated

import (
	"context"
	"net/http"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
)

const (
	// EnvSpaceliftAPIClientTLSCert represents the path to a client certificate for
	// communicating with the Spacelift API endpoint.
	EnvSpaceliftAPIClientTLSCert = client.EnvSpaceliftAPIClientTLSCert

	// EnvSpaceliftAPIClientTLSKey represents the path to a client key for
	// communicating with the Spacelift API endpoint.
	EnvSpaceliftAPIClientTLSKey = client.EnvSpaceliftAPIClientTLSKey

	// EnvSpaceliftAPIClientCA represents the path to a CA bundle for
	// verifying Spacelift API endpoint.
	EnvSpaceliftAPIClientCA = client.EnvSpaceliftAPIClientCA
)

// Client is the authenticated client that can be used by all CLI commands.
var (
	auth client.Client
	m    sync.Mutex
)

// Client returns the authenticated client.
//
// This is an unfortunate global which we have to lock for MCP.
// TODO: Refactor this to not use a global.
func Client() client.Client {
	m.Lock()
	defer m.Unlock()

	return auth
}

// Ensure is a way of ensuring that the Client exists, and it meant to be used
// as a Before action for commands that need it.
//
// You can also use it diretly to refresh the client.
func Ensure(ctx context.Context, _ *cli.Command) (context.Context, error) {
	m.Lock()
	defer m.Unlock()

	httpClient, err := HTTPClient()
	if err != nil {
		return ctx, err
	}

	sess, err := session.New(ctx, httpClient)
	if err != nil {
		return ctx, err
	}

	auth = client.New(httpClient, sess)

	return ctx, nil
}

// HTTPClient returns the shared HTTP client, configured with the TLS settings
// from the environment and the proxy authentication from the current profile.
// Commands that create their own sessions need it to honor the same settings
// as commands using Ensure.
func HTTPClient() (*http.Client, error) {
	return client.GetHTTPClient()
}
