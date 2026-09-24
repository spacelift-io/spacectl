package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var oidcHTTPClient = &http.Client{Timeout: 60 * time.Second}

const (
	oidcProviderGitHubActions = "github-actions"

	envGitHubActionsIDTokenRequestURL   = "ACTIONS_ID_TOKEN_REQUEST_URL"
	envGitHubActionsIDTokenRequestToken = "ACTIONS_ID_TOKEN_REQUEST_TOKEN" // #nosec G101
)

// githubOIDCFromEnvironment reads the OIDC settings for an API key. It returns
// nil when SPACELIFT_API_KEY_OIDC_PROVIDER is unset, so the key uses its secret.
func githubOIDCFromEnvironment(lookup func(string) (string, bool), endpoint string) (*githubOIDC, error) {
	provider, _ := lookup(EnvSpaceliftAPIKeyOIDCProvider)
	audience, _ := lookup(EnvSpaceliftAPIKeyOIDCAudience)
	keySecret, _ := lookup(EnvSpaceliftAPIKeySecret)

	switch {
	case provider == "" && audience != "":
		return nil, fmt.Errorf("%s is set but %s is not; set it to %q to mint an OIDC token", EnvSpaceliftAPIKeyOIDCAudience, EnvSpaceliftAPIKeyOIDCProvider, oidcProviderGitHubActions)
	case provider == "":
		return nil, nil
	case keySecret != "":
		return nil, fmt.Errorf("set either %s or %s, not both", EnvSpaceliftAPIKeySecret, EnvSpaceliftAPIKeyOIDCProvider)
	case provider != oidcProviderGitHubActions:
		return nil, fmt.Errorf("unsupported %s %q, supported values: %s", EnvSpaceliftAPIKeyOIDCProvider, provider, oidcProviderGitHubActions)
	}

	requestURL, _ := lookup(envGitHubActionsIDTokenRequestURL)
	requestToken, _ := lookup(envGitHubActionsIDTokenRequestToken)
	if requestURL == "" || requestToken == "" {
		return nil, fmt.Errorf("%s=%s needs %s and %s, which GitHub Actions only sets when the job has `permissions: id-token: write`",
			EnvSpaceliftAPIKeyOIDCProvider, provider, envGitHubActionsIDTokenRequestURL, envGitHubActionsIDTokenRequestToken)
	}

	if audience == "" {
		u, err := url.Parse(endpoint)
		if err != nil || u.Hostname() == "" {
			return nil, fmt.Errorf("could not derive the OIDC audience from %s %q; set %s", EnvSpaceliftAPIKeyEndpoint, endpoint, EnvSpaceliftAPIKeyOIDCAudience)
		}
		audience = u.Hostname()
	}

	return &githubOIDC{requestURL: requestURL, requestToken: requestToken, audience: audience}, nil
}

type githubOIDC struct {
	requestURL, requestToken, audience string
}

// session builds an API key session whose secret is a GitHub OIDC token
// minted on every exchange.
func (o *githubOIDC) session(ctx context.Context, client *http.Client, endpoint, keyID string) (Session, error) {
	out := &apiKey{
		apiToken:   apiToken{client: client, endpoint: endpoint, timer: time.Now},
		keyID:      keyID,
		githubOIDC: o,
	}

	if err := out.exchange(ctx); err != nil {
		return nil, fmt.Errorf("using a %s OIDC token with audience %q as the API key secret: %w", oidcProviderGitHubActions, o.audience, err)
	}

	return out, nil
}

// mint requests an ID token from the GitHub Actions runner.
func (o *githubOIDC) mint(ctx context.Context) (string, error) {
	u, err := url.Parse(o.requestURL)
	if err != nil {
		return "", fmt.Errorf("could not parse %s: %w", envGitHubActionsIDTokenRequestURL, err)
	}

	query := u.Query()
	query.Set("audience", o.audience)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("could not build the GitHub Actions OIDC token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.requestToken)

	resp, err := oidcHTTPClient.Do(req)
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
