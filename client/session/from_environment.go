package session

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	// EnvSpaceliftAPIEndpoint represents the name of the environment variable
	// pointing to the Spacelift API endpoint.
	//
	// Deprecated
	EnvSpaceliftAPIEndpoint = "SPACELIFT_API_ENDPOINT"

	// EnvSpaceliftAPIKeyEndpoint represents the name of the environment variable
	// pointing to the Spacelift API endpoint.
	EnvSpaceliftAPIKeyEndpoint = "SPACELIFT_API_KEY_ENDPOINT" //nolint: gosec

	// EnvSpaceliftAPIKeyID represents the name of the environment variable
	// pointing to the Spacelift API key ID.
	EnvSpaceliftAPIKeyID = "SPACELIFT_API_KEY_ID" //nolint: gosec

	// EnvSpaceliftAPIKeySecret represents the name of the environment variable
	// pointing to the Spacelift API key secret.
	EnvSpaceliftAPIKeySecret = "SPACELIFT_API_KEY_SECRET" // #nosec G101

	// EnvSpaceliftAPIToken represents the name of the environment variable
	// pointing to the Spacelift API token.
	EnvSpaceliftAPIToken = "SPACELIFT_API_TOKEN" // #nosec G101

	// EnvSpaceliftAPIGitHubToken represents the name of the environment variable
	// pointing to the GitHub access token used to get the Spacelift API token.
	EnvSpaceliftAPIGitHubToken = "SPACELIFT_API_GITHUB_TOKEN" // #nosec G101

	// EnvSpaceliftAPIPreferredMethod represents the name of the environment variable
	// that specifies the preferred authentication method. Valid values: AuthMethodToken, AuthMethodGitHub, AuthMethodAPIKey.
	// If not set, the default priority is: token -> github -> apikey.
	EnvSpaceliftAPIPreferredMethod = "SPACELIFT_API_PREFERRED_METHOD"
)

const (
	authMethodToken  = "token"
	authMethodGitHub = "github"
	authMethodAPIKey = "apikey"
)

var (
	errEnvSpaceliftAPIKeyID       = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeyID)
	errEnvSpaceliftAPIKeySecret   = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeySecret)
	errEnvSpaceliftAPIKeyEndpoint = fmt.Errorf("%s missing from the environment", EnvSpaceliftAPIKeyEndpoint)
)

// FromEnvironment creates a Spacelift session from the environment.
func FromEnvironment(ctx context.Context, client *http.Client) func(func(string) (string, bool)) (Session, error) {
	return func(lookup func(string) (string, bool)) (Session, error) {
		if lookup == nil {
			lookup = os.LookupEnv
		}

		preferredMethod, _ := lookup(EnvSpaceliftAPIPreferredMethod)
		preferredMethod = strings.ToLower(strings.TrimSpace(preferredMethod))

		if preferredMethod != "" {
			return tryAuthMethod(ctx, client, preferredMethod, lookup)
		}

		var lastErr error
		for _, method := range []string{authMethodToken, authMethodGitHub, authMethodAPIKey} {
			session, err := tryAuthMethod(ctx, client, method, lookup)
			if err != nil {
				lastErr = err
			}
			if session != nil {
				return session, nil
			}
		}
		return nil, lastErr
	}
}

func tryAuthMethod(ctx context.Context, client *http.Client, method string, lookup func(string) (string, bool)) (Session, error) {
	switch method {
	case authMethodToken:
		if token, ok := lookup(EnvSpaceliftAPIToken); ok && token != "" {
			return FromAPIToken(ctx, client)("", token)
		}
		return nil, errors.New("SPACELIFT_API_TOKEN not set in environment")

	case authMethodGitHub:
		endpoint, err := getEndpoint(lookup)
		if err != nil {
			return nil, err
		}
		if gitHubToken, ok := lookup(EnvSpaceliftAPIGitHubToken); ok && gitHubToken != "" {
			return FromGitHubToken(ctx, client)(endpoint, gitHubToken)
		}
		return nil, errors.New("SPACELIFT_API_GITHUB_TOKEN not set in environment")

	case authMethodAPIKey:
		endpoint, err := getEndpoint(lookup)
		if err != nil {
			return nil, err
		}
		keyID, ok := lookup(EnvSpaceliftAPIKeyID)
		if !ok || keyID == "" {
			return nil, errEnvSpaceliftAPIKeyID
		}
		keySecret, ok := lookup(EnvSpaceliftAPIKeySecret)
		if !ok || keySecret == "" {
			return nil, errEnvSpaceliftAPIKeySecret
		}
		return FromAPIKey(ctx, client)(endpoint, keyID, keySecret)

	default:
		return nil, fmt.Errorf("no such method %q", method)
	}
}

func getEndpoint(lookup func(string) (string, bool)) (string, error) {
	endpoint, deprecated, err := endpointFromEnvironment(lookup)
	if err != nil {
		return "", err
	}
	if deprecated {
		fmt.Printf("Environment variable %q is deprecated, please use %q\n", EnvSpaceliftAPIEndpoint, EnvSpaceliftAPIKeyEndpoint)
	}
	return endpoint, nil
}

func endpointFromEnvironment(lookup func(string) (string, bool)) (string, bool, error) {
	endpoint, ok := lookup(EnvSpaceliftAPIKeyEndpoint)
	if !ok || endpoint == "" {
		// Keep backwards compatibility with older version of spacectl.
		endpoint, ok = lookup(EnvSpaceliftAPIEndpoint)
		if !ok {
			return "", false, errEnvSpaceliftAPIKeyEndpoint
		}
		return strings.TrimSuffix(endpoint, "/"), true, nil
	}
	return strings.TrimSuffix(endpoint, "/"), false, nil
}

func EnvironmentConfigured() bool {
	return environmentConfigured(os.LookupEnv)
}

func environmentConfigured(lookup func(string) (string, bool)) bool {
	if token, ok := lookup(EnvSpaceliftAPIToken); ok && token != "" {
		return true
	}

	endpoint, _, err := endpointFromEnvironment(lookup)
	if err != nil || endpoint == "" {
		return false
	}

	if token, ok := lookup(EnvSpaceliftAPIGitHubToken); ok && token != "" {
		return true
	}

	keyID, idOK := lookup(EnvSpaceliftAPIKeyID)
	keySecret, secretOK := lookup(EnvSpaceliftAPIKeySecret)

	return idOK && keyID != "" && secretOK && keySecret != ""
}
