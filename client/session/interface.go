package session

import (
	"context"
	"log"
)

// Session is an abstraction around session creation based on credentials from
// various sources.
type Session interface {
	BearerToken(ctx context.Context) (string, error)
	Endpoint() string
}

// Must provides a helper that either creates a Session or dies trying.
func Must(out Session, err error) Session {
	if err != nil {
		log.Fatalf("Could not create a Spacelift session: %v", err)
	}

	return out
}
