package internal

import (
	"context"
	"fmt"
	"io"
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

type IgnoreMatcherFn func(filePath string) bool

func GetIgnoreMatcherFn(ctx context.Context, projectRoot *string, ignoreFiles []string) (IgnoreMatcherFn, error) {
	baseDir := "."
	if projectRoot != nil {
		baseDir = *projectRoot
	}

	ignoreFilesByDir, err := discoverIgnoreFiles(baseDir, ignoreFiles)
	if err != nil {
		return nil, err
	}

	customIgnore := ignore.CompileIgnoreLines(".git", ".terraform")

	return func(filePath string) bool {
		cleanPath := filepath.Clean(filePath)

		if customIgnore.MatchesPath(cleanPath) {
			return false
		}

		if isIgnoredByAnyFile(ignoreFilesByDir, cleanPath) {
			return false
		}

		if projectRoot != nil && !isWithinProjectRoot(*projectRoot, cleanPath) {
			return false
		}

		return true
	}, nil
}

type ignoreFileInfo struct {
	ignoreFile *ignore.GitIgnore
	directory  string
}

func discoverIgnoreFiles(baseDir string, ignoreFiles []string) ([]ignoreFileInfo, error) {
	var allIgnoreFiles []ignoreFileInfo

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		for _, ignoreFileName := range ignoreFiles {
			if info.Name() == ignoreFileName {
				ignoreFile, compileErr := ignore.CompileIgnoreFile(path)
				if compileErr != nil && !os.IsNotExist(compileErr) {
					return fmt.Errorf("couldn't compile %s file at %s: %w", ignoreFileName, path, compileErr)
				}
				if ignoreFile != nil {
					directory := filepath.Dir(path)
					allIgnoreFiles = append(allIgnoreFiles, ignoreFileInfo{
						ignoreFile: ignoreFile,
						directory:  directory,
					})
				}
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't walk directory tree to find ignore files: %w", err)
	}

	return allIgnoreFiles, nil
}

func isIgnoredByAnyFile(ignoreFiles []ignoreFileInfo, filePath string) bool {
	for _, info := range ignoreFiles {
		if isWithinDirectory(info.directory, filePath) {
			relPath, err := filepath.Rel(info.directory, filePath)
			if err == nil && info.ignoreFile.MatchesPath(relPath) {
				return true
			}
		}
	}
	return false
}

func isWithinDirectory(directory, filePath string) bool {
	cleanDir := filepath.Clean(directory)
	cleanFile := filepath.Clean(filePath)

	if cleanDir == "." {
		return true
	}

	return cleanFile == cleanDir || strings.HasPrefix(cleanFile, cleanDir+string(filepath.Separator))
}

func isWithinProjectRoot(projectRoot, filePath string) bool {
	rootPath := filepath.Clean(projectRoot)

	if filepath.IsAbs(filePath) {
		absRoot, err := filepath.Abs(rootPath)
		if err != nil {
			return false
		}
		return filePath == absRoot || strings.HasPrefix(filePath, absRoot+string(filepath.Separator))
	}

	return filePath == rootPath || strings.HasPrefix(filePath, rootPath+string(filepath.Separator))
}

// UploadArchive uploads a tarball to the target endpoint and displays a fancy progress bar.
func UploadArchive(ctx context.Context, uploadURL, path string, uploadHeaders map[string]string, showProgress bool) (err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("couldn't stat archive file: %w", err)
	}

	// #nosec G304
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("couldn't open archive file: %w", err)
	}

	var reader io.Reader = f
	if showProgress {
		bar := pb.Full.Start64(stat.Size())
		reader = bar.NewProxyReader(f)
		defer bar.Finish()
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, reader)
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
