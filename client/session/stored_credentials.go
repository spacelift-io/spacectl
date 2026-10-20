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

// EnvSpaceliftProxyAuthToken is the name of the environment variable that,
// when set, supplies the token sent in the Proxy-Authorization header. It
// takes precedence over any proxy_auth configuration in the profile.
const EnvSpaceliftProxyAuthToken = "SPACELIFT_PROXY_AUTH_TOKEN" // #nosec G101

// String returns the string representation of the type.
func (t CredentialsType) String() string {
	return [...]string{"Invalid", "API Key", "GitHub", "API Token"}[t]
}

// ProxyAuthConfig configures a Proxy-Authorization header to be injected on
// outgoing requests. This is useful when Spacelift sits behind a reverse proxy
// (such as GCP Identity-Aware Proxy or Azure AD Application Proxy) that
// requires its own authentication separate from Spacelift's bearer token.
//
// Sources are consulted in this order: the SPACELIFT_PROXY_AUTH_TOKEN
// environment variable, then TokenFile, then Command. The first one that is
// set wins.
type ProxyAuthConfig struct {
	// TokenFile is the path of a file whose contents (trimmed) are used as the
	// proxy auth token. The file is re-read when the proxy rejects the token,
	// so it can be refreshed by an external process.
	TokenFile string `json:"token_file,omitempty"`

	// Command is the command (program + args) to run to obtain a proxy auth
	// token. The token is read from stdout (trimmed). The program must be given
	// as an absolute path, and the configuration file that names it must not be
	// writable by other users.
	Command []string `json:"command,omitempty"`

	// SetupHint is appended to the error reported when the token file or
	// Command is missing, fails, or produces no token. It should tell the user
	// how to install or initialize the auth tool.
	SetupHint string `json:"setup_hint,omitempty"`
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
