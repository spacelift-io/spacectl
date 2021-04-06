package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
)

const (
	// SpaceliftConfigDirectory is the name of the Spacelift config directory.
	SpaceliftConfigDirectory = ".spacelift"

	// CurrentFileName is the name of the "current" Spacelift profile.
	CurrentFileName = "current"

	// CurrentFilePath is the path to the "current" Spacelift profile.
	CurrentFilePath = SpaceliftConfigDirectory + "/" + CurrentFileName
)

// FromFile creates a session from credentials stored in a file.
func FromFile(ctx context.Context, client *http.Client) func(path string) (Session, error) {
	return func(path string) (Session, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("could not open Spacelift credentials from %s: %w", path, err)
		}
		defer file.Close()

		var out StoredCredentials
		if err := json.NewDecoder(file).Decode(&out); err != nil {
			return nil, fmt.Errorf("could not decode Spacelift credentials form %s: %w", path, err)
		}

		return out.Session(ctx, client)
	}
}

// FromCurrentFile creates a session from credentials stored in the default
// current file.
func FromCurrentFile(ctx context.Context, client *http.Client) (Session, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find user home directory: %w", err)
	}

	return FromFile(ctx, client)(path.Join(userHomeDir, CurrentFilePath))
}
