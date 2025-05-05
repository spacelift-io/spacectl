package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/mholt/archiver/v3"
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

// GetIgnoreMatcherFn creates an ignore-matcher for archiving purposes
// This function respects gitignore and terraformignore, and
// optionally if a projectRoot is provided it only include files from this root
func GetIgnoreMatcherFn(ctx context.Context, projectRoot *string, ignoreFiles []string) (IgnoreMatcherFn, error) {
	ignoreList := make([]*ignore.GitIgnore, 0)
	for _, f := range ignoreFiles {
		ignoreFile, err := ignore.CompileIgnoreFile(f)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("couldn't compile %s file: %w", f, err)
		}

		ignoreList = append(ignoreList, ignoreFile)
	}

	projectRootPrefixes := make(map[string]struct{})
	if projectRoot != nil {
		rootPrefix := "."
		projectRootPrefixes[rootPrefix] = struct{}{}
		for _, part := range strings.Split(*projectRoot, string(os.PathSeparator)) {
			rootPrefix = filepath.Join(rootPrefix, part)
			projectRootPrefixes[rootPrefix] = struct{}{}
		}
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
			// We must include all path prefixes of the projectRoot as well.
			if _, ok := projectRootPrefixes[filePath]; ok {
				return true
			}
			// ensure the root only matches the directory and not all other files
			root := strings.TrimSuffix(*projectRoot, "/") + "/"
			if !strings.HasPrefix(filePath, root) {
				return false
			}
		}

		return true
	}, nil
}

// Create a tar.gz of the contents of src at dest. The contents of dest are
// filtered by the matchFn. To speed up processing of large ignored directories
// we also short circuit the file system walk if a directory is ignored.
func CreateArchive(ctx context.Context, src, dest string, matchFn IgnoreMatcherFn) error {
	if !strings.HasSuffix(dest, ".tar.gz") {
		fmt.Errorf(".tar.gz extention required: %s", dest)
	}

	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat %s, %w", src, err)
	}

	tgz := *archiver.DefaultTarGz

	destDir := filepath.Dir(dest)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("%s: mkdir %w", destDir, err)
		}
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer out.Close()

	err = tgz.Create(out)
	if err != nil {
		return fmt.Errorf("creating tgz: %w", err)
	}
	defer tgz.Close()

	base := filepath.Base(dest)
	prefixInArchive := strings.TrimSuffix(base, ".tar.gz")

	return filepath.Walk(src, func(fpath string, info os.FileInfo, err error) error {
		if !matchFn(fpath) {
			if info.Mode().IsDir() {
				return filepath.SkipDir
			}
			if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return nil
			}
		}
		nameInArchive, err := archiver.NameInArchive(srcInfo, src, fpath)
		if err != nil {
			return fmt.Errorf("NameInArchive %s: %w", fpath, err)
		}
		fullNameInArchive := path.Join(prefixInArchive, nameInArchive)
		var file io.ReadCloser
		if info.Mode().IsRegular() {
			file, err = os.Open(fpath)
			if err != nil {
				return fmt.Errorf("%s: open: %w", fpath, err)
			}
			defer file.Close()
		}
		err = tgz.Write(archiver.File{
			FileInfo: archiver.FileInfo{
				FileInfo:   info,
				CustomName: fullNameInArchive,
			},
			FullFilePath: fpath,
			ReadCloser:   file,
		})
		if err != nil {
			return fmt.Errorf("%s: writing: %w", fpath, err)
		}

		return nil
	})
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
