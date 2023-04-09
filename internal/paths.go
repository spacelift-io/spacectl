package internal

import (
	"path/filepath"
)

// ParentDirectory returns the directory of the given `path`.
//
// If a parent directory could be found, then this function returns a non-empty
// string and `true`.
//
// If a parent directory could not be found, this function returns an empty
// string and `false`.
//
// That means this function returns two non-zero values on success, and two
// zero-values on error.
func ParentDirectory(path string) (string, bool) {
	cleanPath := filepath.Clean(path)

	// If `cleanPath` is relative, ensure it doesn't go to ".." or above.
	if !filepath.IsAbs(cleanPath) {
		joinedToRoot := filepath.Join("/", cleanPath)

		relPath, err := filepath.Rel("/", joinedToRoot)
		if err != nil || cleanPath != relPath {
			return "", false
		}
	}

	parentDir := filepath.Dir(cleanPath)
	ok := len(parentDir) < len(cleanPath)
	if !ok {
		return "", false
	}

	return parentDir, true
}

// PathAncestors returns a slice containing both the directory of `initialPath`
// and all its ancestor directories.
func PathAncestors(initialPath string) []string {
	var ancestors []string

	for nextAncestor, ok := ParentDirectory(initialPath); ok; nextAncestor, ok = ParentDirectory(ancestors[len(ancestors)-1]) {
		ancestors = append(ancestors, nextAncestor)
	}

	return ancestors
}
