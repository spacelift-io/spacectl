package internal

import "fmt"

// GPGKey represents a GPG key as seen by the Spacelift API.
type GPGKey struct {
	ID          string  `graphql:"id" json:"id"`
	CreatedAt   int64   `graphql:"createdAt" json:"createdAt"`
	CreatedBy   string  `graphql:"createdBy" json:"createdBy"`
	Description *string `graphql:"description" json:"description"`
	Name        string  `graphql:"name" json:"name"`
	RevokedAt   *int64  `graphql:"revokedAt" json:"revokedAt"`
	RevokedBy   *string `graphql:"revokedBy" json:"revokedBy"`
	UpdatedAt   int64   `graphql:"updatedAt" json:"updatedAt"`
}

// Row returns a row representation of the GPG key.
func (g GPGKey) Row() []string {
	return []string{
		g.ID,
		g.Name,
		fmt.Sprintf("%t", g.RevokedAt != nil),
	}
}

// GPGKeys is a list of GPG keys.
type GPGKeys []GPGKey

// Headers returns a list of headers for the GPG keys table.
func (g GPGKeys) Headers() []string {
	return []string{"ID (Fingerprint)", "Name", "Revoked"}
}
