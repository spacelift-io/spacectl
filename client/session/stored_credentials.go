package session

import (
	"context"
	"fmt"
	"net/http"
)

// CredentialsType represents the type of credentials being used.
type CredentialsType uint

const (
	// CredentialsTypeInvalid represents an invalid zero value for the
	// CredentialsType.
	CredentialsTypeInvalid CredentialsType = iota

	// CredentialsTypeAPIKey represents credentials stored as an API key
	// id-secret pair.
	CredentialsTypeAPIKey

	// CredentialsTypeGitHubToken represents credentials stored as a GitHub
	// access token.
	CredentialsTypeGitHubToken

	// CredentialsTypeAPIToken represents credentials stored as a JWT
	// access token.
	CredentialsTypeAPIToken
)

// String returns the string representation of the type.
func (t CredentialsType) String() string {
	return [...]string{"Invalid", "API Key", "GitHub", "API Token"}[t]
}

// ProxyAuthConfig configures a Proxy-Authorization header to be injected on
// outgoing requests. This is useful when Spacelift sits behind a reverse proxy
// (such as GCP Identity-Aware Proxy or Azure AD Application Proxy) that
// requires its own authentication separate from Spacelift's bearer token.
type ProxyAuthConfig struct {
	// Command is the command (program + args) to run to obtain a proxy auth
	// token. The token is read from stdout (trimmed).
	Command []string `json:"command"`

	// SetupHint is shown when Command is missing or returns empty AND the
	// server rejects the request. It should tell the user how to install or
	// initialize the auth tool.
	SetupHint string `json:"setup_hint,omitempty"`

	// RefreshHint is shown when Command returns a token but the server
	// still rejects it (e.g. the token has expired).
	RefreshHint string `json:"refresh_hint,omitempty"`
}

// StoredCredentials is a filesystem representation of the credentials.
type StoredCredentials struct {
	Type        CredentialsType  `json:"type,omitempty"`
	Endpoint    string           `json:"endpoint,omitempty"`
	AccessToken string           `json:"access_token,omitempty"`
	KeyID       string           `json:"key_id,omitempty"`
	KeySecret   string           `json:"key_secret,omitempty"`
	ProxyAuth   *ProxyAuthConfig `json:"proxy_auth,omitempty"`
}

// Session creates a Spacelift Session from stored credentials.
func (s *StoredCredentials) Session(ctx context.Context, client *http.Client) (Session, error) {
	switch s.Type {
	case CredentialsTypeAPIKey:
		return FromAPIKey(ctx, client)(s.Endpoint, s.KeyID, s.KeySecret)
	case CredentialsTypeGitHubToken:
		return FromGitHubToken(ctx, client)(s.Endpoint, s.AccessToken)
	case CredentialsTypeAPIToken:
		return FromAPIToken(ctx, client)(s.Endpoint, s.AccessToken)
	default:
		return nil, fmt.Errorf("unexpected credentials type: %d", s.Type)
	}
}
