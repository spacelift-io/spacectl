package session

import (
	"context"
	"log/slog"
	"os"
)

// Session is an abstraction around session creation based on credentials from
// various sources.
type Session interface {
	BearerToken(ctx context.Context) (string, error)
	Endpoint() string
	Type() CredentialsType
}

// Must provides a helper that either creates a Session or dies trying.
func Must(out Session, err error) Session {
	if err != nil {
		slog.Error("Could not create a Spacelift session", "err", err)
		os.Exit(1)
	}

	return out
}
