package session

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	// EnvSpaceliftAPIEndpoint represents the name of the environment variable
	// pointing to the Spacelift API endpoint.
	//
	// Deprecated
	EnvSpaceliftAPIEndpoint = "SPACELIFT_API_ENDPOINT"

	// EnvSpaceliftAPIKeyEndpoint represents the name of the environment variable
	// pointing to the Spacelift API endpoint.
	EnvSpaceliftAPIKeyEndpoint = "SPACELIFT_API_KEY_ENDPOINT"

	// EnvSpaceliftAPIKeyID represents the name of the environment variable
	// pointing to the Spacelift API key ID.
	EnvSpaceliftAPIKeyID = "SPACELIFT_API_KEY_ID"

	// EnvSpaceliftAPIKeySecret represents the name of the environment variable
	// pointing to the Spacelift API key secret.
	EnvSpaceliftAPIKeySecret = "SPACELIFT_API_KEY_SECRET" // #nosec G101

	// EnvSpaceliftAPIToken represents the name of the environment variable
	// pointing to the Spacelift API token.
	EnvSpaceliftAPIToken = "SPACELIFT_API_TOKEN" // #nosec G101

	// EnvSpaceliftAPIGitHubToken represents the name of the environment variable
	// pointing to the GitHub access token used to get the Spacelift API token.
	EnvSpaceliftAPIGitHubToken = "SPACELIFT_API_GITHUB_TOKEN" // #nosec G101
)

var (
	ErrEnvSpaceliftAPIKeyID       = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeyID)
	ErrEnvSpaceliftAPIKeySecret   = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeySecret)
	ErrEnvSpaceliftAPIKeyEndpoint = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeyEndpoint)
)

// FromEnvironment creates a Spacelift session from the environment.
func FromEnvironment(ctx context.Context, client *http.Client) func(func(string) (string, bool)) (Session, error) {
	return func(lookup func(string) (string, bool)) (Session, error) {
		if lookup == nil {
			lookup = os.LookupEnv
		}

		if token, ok := lookup(EnvSpaceliftAPIToken); ok && token != "" {
			return FromAPIToken(ctx, client)(token)
		}

		endpoint, ok := lookup(EnvSpaceliftAPIKeyEndpoint)
		if !ok {
			// Keep backwards compatibility with older version of spacectl.
			endpoint, ok = lookup(EnvSpaceliftAPIEndpoint)
			if !ok {
				return nil, ErrEnvSpaceliftAPIKeyEndpoint
			}
			fmt.Printf("Environment variable %q is deprecated, please use %q\n", EnvSpaceliftAPIEndpoint, EnvSpaceliftAPIKeyEndpoint)
		}
		endpoint = strings.TrimSuffix(endpoint, "/")

		if gitHubToken, ok := lookup(EnvSpaceliftAPIGitHubToken); ok && gitHubToken != "" {
			return FromGitHubToken(ctx, client)(endpoint, gitHubToken)
		}

		keyID, ok := lookup(EnvSpaceliftAPIKeyID)
		if !ok || keyID == "" {
			return nil, ErrEnvSpaceliftAPIKeyID
		}

		keySecret, ok := lookup(EnvSpaceliftAPIKeySecret)
		if !ok || keySecret == "" {
			return nil, ErrEnvSpaceliftAPIKeySecret
		}

		return FromAPIKey(ctx, client)(endpoint, keyID, keySecret)
	}
}
