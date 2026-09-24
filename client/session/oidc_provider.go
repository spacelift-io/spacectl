package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	oidcProviderGitHubActions = "github-actions"

	envGitHubActionsIDTokenRequestURL   = "ACTIONS_ID_TOKEN_REQUEST_URL"
	envGitHubActionsIDTokenRequestToken = "ACTIONS_ID_TOKEN_REQUEST_TOKEN" // #nosec G101
)

// fromOIDCProvider builds an API key session whose secret is an OIDC token
// minted from the CI provider on every exchange.
func fromOIDCProvider(ctx context.Context, client *http.Client, lookup func(string) (string, bool), endpoint, keyID, provider, audience string) (Session, error) {
	if audience == "" {
		u, err := url.Parse(endpoint)
		if err != nil || u.Hostname() == "" {
			return nil, fmt.Errorf("could not derive the OIDC audience from %s %q; set %s", EnvSpaceliftAPIKeyEndpoint, endpoint, EnvSpaceliftAPIKeyOIDCAudience)
		}
		audience = u.Hostname()
	}

	source, err := oidcTokenSource(provider, audience, lookup)
	if err != nil {
		return nil, err
	}

	session, err := fromAPIKeySecretSource(ctx, client, endpoint, keyID, source)
	if err != nil {
		return nil, fmt.Errorf("using a %s OIDC token with audience %q as the API key secret: %w", provider, audience, err)
	}

	return session, nil
}

func oidcTokenSource(provider, audience string, lookup func(string) (string, bool)) (func(context.Context) (string, error), error) {
	switch provider {
	case oidcProviderGitHubActions:
		requestURL, _ := lookup(envGitHubActionsIDTokenRequestURL)
		requestToken, _ := lookup(envGitHubActionsIDTokenRequestToken)
		if requestURL == "" || requestToken == "" {
			return nil, fmt.Errorf("%s=%s needs %s and %s, which GitHub Actions only sets when the job has `permissions: id-token: write`",
				EnvSpaceliftAPIKeyOIDCProvider, provider, envGitHubActionsIDTokenRequestURL, envGitHubActionsIDTokenRequestToken)
		}

		return func(ctx context.Context) (string, error) {
			return mintGitHubActionsToken(ctx, requestURL, requestToken, audience)
		}, nil

	default:
		return nil, fmt.Errorf("unsupported %s %q, supported values: %s", EnvSpaceliftAPIKeyOIDCProvider, provider, oidcProviderGitHubActions)
	}
}

// mintGitHubActionsToken requests an ID token from the GitHub Actions runner.
// It uses http.DefaultClient because the session's client may trust only the
// Spacelift CA from SPACELIFT_API_TLS_CA, which would fail against GitHub.
func mintGitHubActionsToken(ctx context.Context, requestURL, requestToken, audience string) (string, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("could not parse %s: %w", envGitHubActionsIDTokenRequestURL, err)
	}

	query := u.Query()
	query.Set("audience", audience)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("could not build the GitHub Actions OIDC token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not request a GitHub Actions OIDC token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not request a GitHub Actions OIDC token: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("could not decode the GitHub Actions OIDC token response: %w", err)
	}
	if body.Value == "" {
		return "", fmt.Errorf("GitHub Actions returned an empty OIDC token")
	}

	return body.Value, nil
}
