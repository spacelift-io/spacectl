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
)

// StoredCredentials is a filesystem representation of the credentials.
type StoredCredentials struct {
	Type        CredentialsType `json:"type,omitempty"`
	Endpoint    string          `json:"endpoint,omitempty"`
	AccessToken string          `json:"access_token,omitempty"`
	KeyID       string          `json:"key_id,omitempty"`
	KeySecret   string          `json:"key_secret,omitempty"`
}

// Session creates a Spacelift Session from stored credentials.
func (s *StoredCredentials) Session(ctx context.Context, client *http.Client) (Session, error) {
	switch s.Type {
	case CredentialsTypeAPIKey:
		return FromAPIKey(ctx, client)(s.Endpoint, s.KeyID, s.KeySecret)
	case CredentialsTypeGitHubToken:
		return FromGitHubToken(ctx, client)(s.Endpoint, s.AccessToken)
	default:
		return nil, fmt.Errorf("unexpected credentials type: %d", s.Type)
	}
}
