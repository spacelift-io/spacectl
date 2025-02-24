package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb/v3"
	ignore "github.com/sabhiram/go-gitignore"
)

// MoveToRepositoryRoot moves the current workdir to the git repository root.
func MoveToRepositoryRoot() error {
	startCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get current working directory: %w", err)
	}
	for {
		if _, err := os.Stat(".git"); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("couldn't stat .git directory: %w", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("couldn't get current working directory: %w", err)
		}

		parent := filepath.Dir(cwd)

		if parent == cwd {
			fmt.Println("Couldn't find repository root, using current directory.")
			if err := os.Chdir(startCwd); err != nil {
				return fmt.Errorf("couldn't set current working directory: %w", err)
			}
			return nil
		}

		if err := os.Chdir(parent); err != nil {
			return fmt.Errorf("couldn't set current working directory: %w", err)
		}
	}
}

// GetIgnoreMatcherFn creates an ignore-matcher for archiving purposes
// This function respects gitignore and terraformignore, and
// optionally if a projectRoot is provided it only include files from this root
func GetIgnoreMatcherFn(ctx context.Context, projectRoot *string, ignoreFiles []string) (func(filePath string) bool, error) {
	ignoreList := make([]*ignore.GitIgnore, 0)
	for _, f := range ignoreFiles {
		ignoreFile, err := ignore.CompileIgnoreFile(f)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("couldn't compile %s file: %w", f, err)
		}

		ignoreList = append(ignoreList, ignoreFile)
	}

	customignore := ignore.CompileIgnoreLines(".git", ".terraform")
	return func(filePath string) bool {
		if customignore.MatchesPath(filePath) {
			return false
		}

		for _, v := range ignoreList {
			if v != nil && v.MatchesPath(filePath) {
				return false
			}
		}

		if projectRoot != nil {
			// ensure the root only matches the directory and not all other files
			root := strings.TrimSuffix(*projectRoot, "/") + "/"
			if !strings.HasPrefix(filePath, root) {
				return false
			}
		}

		return true
	}, nil
}

// UploadArchive uploads a tarball to the target endpoint and displays a fancy progress bar.
func UploadArchive(ctx context.Context, uploadURL, path string, uploadHeaders map[string]string) (err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("couldn't stat archive file: %w", err)
	}

	// #nosec G304
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("couldn't open archive file: %w", err)
	}

	bar := pb.Full.Start64(stat.Size())
	barReader := bar.NewProxyReader(f)
	defer bar.Finish()

	req, err := http.NewRequest(http.MethodPut, uploadURL, barReader)
	if err != nil {
		return fmt.Errorf("couldn't create upload request: %w", err)
	}
	req.ContentLength = stat.Size()
	req = req.WithContext(ctx)

	for k, v := range uploadHeaders {
		req.Header.Set(k, v)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't upload workspace: %w", err)
	}
	defer response.Body.Close()
	if code := response.StatusCode; code != http.StatusOK && code != http.StatusCreated {
		return fmt.Errorf("unexpected response code when uploading workspace: %d", code)
	}

	return nil
}
