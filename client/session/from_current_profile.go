package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// FromCurrentProfile creates a session from credentials stored in the currently selected profile.
func FromCurrentProfile(ctx context.Context, client *http.Client) (Session, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find user home directory: %w", err)
	}

	manager, err := NewProfileManager(filepath.Join(userHomeDir, SpaceliftConfigDirectory))
	if err != nil {
		return nil, fmt.Errorf("could not create profile manager: %w", err)
	}

	currentProfile := manager.Current()
	if currentProfile == nil {
		return nil, errors.New("no current profile is set - please login first")
	}

	return currentProfile.Credentials.Session(ctx, client)
}
