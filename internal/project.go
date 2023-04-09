package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

var (
	ErrNotFound              error = errors.New("not found")
	ErrUnknownIgnoreFileType error = errors.New("unknown ignore file type")
)

// Project is a convenience struct that allows doing project-wide operations.
//
// The methods of this struct are NOT thread-safe.
type Project struct {
	RootDirectory string

	// ignoreFilesByDirectory contains all known ignore files.
	//
	// The key of the map is a path to a directory, relative to
	// `.RootDirectory`; the value is a slice containing all ignore files from
	// that directory.
	//
	// This map is initialized by calling the `UpdateIgnoreFiles` method.
	//
	// This is a map because, to know whether a file is ignore or not, we only
	// need to check the ignore files located in the same directory of that
	// file, and any ancestor directories (i.e. never sibling directories).
	ignoreFilesByDirectory map[string][]IgnoreFile
}

// ToRelativePath receives an absolute path, and returns a path relative to the
// root of the project.
func (p *Project) ToRelativePath(absPath string) (string, error) {
	return filepath.Rel(p.RootDirectory, absPath)
}

// ToAbsolutePath receives a path relative to the project root, and returns an
// absolute path.
func (p *Project) ToAbsolutePath(pathRelativeToProjectRoot string) string {
	absPath := filepath.Join(p.RootDirectory, pathRelativeToProjectRoot)

	return absPath
}

// ArchiveMatcher returns a function that, when called with a path, returns
// `true` if the path can be archived, or `false` if the path has been ignored.
//
// The returned function expects receiving a path relative to `referencePath`.
//
// This method expects `UpdateIgnoreFiles` to have been called already.
func (p *Project) ArchiveMatcher(referencePath string) func(path string) bool {
	absMatcherRoot := p.ToAbsolutePath(referencePath)

	return func(pathToMatch string) bool {
		absPathToMatch := filepath.Join(absMatcherRoot, pathToMatch)
		pathRelativeToProjectRoot, err := p.ToRelativePath(absPathToMatch)
		if err != nil {
			return false
		}

		isIgnored, err := p.PathIsIgnored(pathRelativeToProjectRoot)
		if err != nil {
			return false
		}
		if isIgnored {
			return false
		}

		// Ensure the root only matches the directory and not all other files.
		root := strings.TrimSuffix(p.RootDirectory, "/") + "/"
		if !strings.HasPrefix(pathRelativeToProjectRoot, root) {
			return false
		}

		return true
	}
}

func (p *Project) clearIgnoreFiles() {
	p.ignoreFilesByDirectory = make(map[string][]IgnoreFile)
}

// addIgnoreFile receives the path to an ignore file, relative to the project
// root, and registers it in the `*Project` struct.
//
// This method is not thread safe.
func (p *Project) addIgnoreFile(pathRelativeToProjectRoot string) error {
	absPath := filepath.Join(p.RootDirectory, pathRelativeToProjectRoot)

	directory, name := filepath.Split(pathRelativeToProjectRoot)
	if _, ok := isIgnoreFileName[name]; ok {
		instance, err := ignore.CompileIgnoreFile(absPath)
		if err != nil {
			return fmt.Errorf("could not create instance for ignore file: %w", err)
		}

		var ignoreFile IgnoreFile = &gitignoreFile{
			instance: instance,
		}

		p.ignoreFilesByDirectory[directory] = append(p.ignoreFilesByDirectory[directory], ignoreFile)
	} else {
		return fmt.Errorf("unsupported ignore file %#v: %w", absPath, ErrUnknownIgnoreFileType)
	}

	return nil
}

// UpdateIgnoreFiles finds all ignore files in the project, starting at the
// project's root directory, and recursively traversing all subdirectories.
func (p *Project) UpdateIgnoreFiles() error {
	p.clearIgnoreFiles()

	err := filepath.Walk(".", func(pathRelativeToProjectRoot string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := fi.Name()
		if fi.Mode().IsRegular() {
			if _, ok := isIgnoreFileName[name]; ok {
				return p.addIgnoreFile(pathRelativeToProjectRoot)
			}
		} else if fi.IsDir() {
			if _, ok := skipDirs[name]; ok {
				return filepath.SkipDir
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error updating ignore files: %w", err)
	}

	return nil
}

// PathIsIgnored returns `true` if the given path is ignored by any of the
// project's ignore files.
func (p *Project) PathIsIgnored(pathRelativeToProjectRoot string) (bool, error) {
	var err error

	name := filepath.Base(pathRelativeToProjectRoot)
	if _, ok := alwaysIgnoreName[name]; ok {
		return true, nil
	}

	ancestors := PathAncestors(pathRelativeToProjectRoot)
	for _, directory := range ancestors {
		ignoreFiles, ok := p.ignoreFilesByDirectory[directory]
		if !ok {
			continue
		}

		abs := filepath.Join(p.RootDirectory, pathRelativeToProjectRoot)
		abs, err = filepath.Abs(abs)
		if err != nil {
			return false, fmt.Errorf("could not find absolute path for %#v: %w", abs, err)
		}

		pathRelativeToIgnoreFile, err := filepath.Rel(directory, abs)
		if err != nil {
			return false, fmt.Errorf("could not convert to relative path: %#v: %w", abs, err)
		}

		for _, ignoreFile := range ignoreFiles {
			if ignoreFile.Matches(pathRelativeToIgnoreFile) {
				return true, nil
			}
		}
	}

	return false, nil
}

// IsDirectory returns whether the given path is a directory or not.
//
// It is just a convenience function.
func IsDirectory(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("could not stat %#v: %w", path, err)
	}

	if !fi.IsDir() {
		return false, nil
	}

	return true, nil
}

// FindFirstExistingDirectory receives a slice of paths, and returns the first
// of them that is an existing directory.
//
// If none of the given candidates is an existing directory, it returns
// `ErrNotFound`.
func FindFirstExistingDirectory(candidates []string) (string, error) {
	for _, candidate := range candidates {
		isDirectory, err := IsDirectory(candidate)
		if err != nil {
			return "", fmt.Errorf("could not find first existing directory: %w", err)
		}

		if isDirectory {
			return candidate, nil
		}
	}

	return "", ErrNotFound
}

// FindProjectRoot tries to find the root directory of the project.
//
// It receives a path to any file or directory inside the project.
func FindProjectRoot(pathInProject string) (string, error) {
	var possibleGitPaths []string
	pathIsDirectory, err := IsDirectory(pathInProject)
	if err != nil {
		return "", fmt.Errorf("could not find project root: %w", err)
	}

	if pathIsDirectory {
		cleanPath := filepath.Clean(pathInProject)
		cleanPath = filepath.Join(cleanPath, ".git")
		possibleGitPaths = append(possibleGitPaths, cleanPath)
	}

	ancestors := PathAncestors(pathInProject)
	for _, directory := range ancestors {
		possiblePath := filepath.Join(directory, ".git")
		possibleGitPaths = append(possibleGitPaths, possiblePath)
	}

	projectRoot, err := FindFirstExistingDirectory(possibleGitPaths)
	if err != nil {
		return "", fmt.Errorf("could not find project root: %w", err)
	}

	return projectRoot, nil
}
