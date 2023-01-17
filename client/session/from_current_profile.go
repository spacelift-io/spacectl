package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// FromCurrentProfile creates a session from credentials stored in the currently selected profile.
func FromCurrentProfile(ctx context.Context, client *http.Client) (Session, error) {
	manager, err := UserProfileManager()
	if err != nil {
		return nil, fmt.Errorf("could not access profile manager: %w", err)
	}

	currentProfile := manager.Current()
	if currentProfile == nil {
		return nil, errors.New("no current profile is set - please login first")
	}

	return currentProfile.Credentials.Session(ctx, client)
}
