package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mholt/archiver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIgnoreMatcherFnHierarchical(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spacectl-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	dirs := []string{
		"src",
		"src/components",
		"src/utils",
		"tests",
		"tests/unit",
		"docs",
		"build",
		"node_modules",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	testFiles := map[string]string{
		"README.md":      "# Test Project",
		"package.json":   `{"name": "test"}`,
		"secret.env":     "SECRET=value",
		"config.yaml":    "config: test",
		"included.cache": "cache content",

		"src/main.go":             "package main",
		"src/temp.log":            "log content",
		"src/components/comp.go":  "package components",
		"src/components/test.tmp": "temp file",
		"src/utils/util.go":       "package utils",
		"src/utils/debug.log":     "debug log",
		"src/excluded.env":        "excluded env",

		"tests/main_test.go":      "package main",
		"tests/temp.cache":        "cache content",
		"tests/unit/unit_test.go": "package unit",
		"tests/unit/coverage.out": "coverage data",

		"build/output.bin": "binary content",
		"build/temp.o":     "object file",

		"node_modules/lib.js": "library code",

		"docs/readme.txt": "documentation",
		"docs/temp.md":    "temp doc",
	}

	for path, content := range testFiles {
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0644) //nolint: gosec
		require.NoError(t, err)
	}

	ignoreFiles := map[string]string{
		".gitignore": `# Root level ignores
*.env
node_modules/
build/
`,

		".terraformignore": `# Terraform ignores
*.tfstate
*.tfplan
.terraform/
`,

		"src/.gitignore": `# Source level ignores
*.log
*.tmp
`,

		"tests/.gitignore": `# Test level ignores  
*.cache
*.out
`,

		"docs/.terraformignore": `# Docs level ignores
temp.*
`,
	}

	for path, content := range ignoreFiles {
		err := os.WriteFile(path, []byte(content), 0644) //nolint: gosec
		require.NoError(t, err)
	}

	ctx := context.Background()
	ignoreFileNames := []string{".gitignore", ".terraformignore"}

	matchFn, err := GetIgnoreMatcherFn(ctx, nil, ignoreFileNames, false)
	require.NoError(t, err)

	testCases := map[string]bool{
		"README.md":     true,
		"package.json":  true,
		"secret.env":    false,
		"config.yaml":   true,
		"included.yaml": true,

		"src/main.go":             true,
		"src/temp.log":            false,
		"src/components/comp.go":  true,
		"src/components/test.tmp": false,
		"src/utils/util.go":       true,
		"src/utils/debug.log":     false,
		"src/excluded.env":        false,

		"tests/main_test.go":      true,
		"tests/temp.cache":        false,
		"tests/unit/unit_test.go": true,
		"tests/unit/coverage.out": false,

		"build/output.bin": false,
		"build/temp.o":     false,

		"node_modules/lib.js": false,

		"docs/readme.txt": true,
		"docs/temp.md":    false,

		".git/config":      false,
		".terraform/state": false,
	}

	for filePath, shouldInclude := range testCases {
		result := matchFn(filePath)
		assert.Equal(t, shouldInclude, result,
			"File %q: expected included=%v, got included=%v", filePath, shouldInclude, result)
	}
}

func TestGetIgnoreMatcherFnWithArchiver(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spacectl-archiver-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	testStructure := map[string]string{
		"main.go":          "package main",
		"config.yaml":      "config: value",
		"secret.env":       "SECRET=test",
		"included.log":     "SECRET=test",
		"src/app.go":       "package src",
		"src/debug.log":    "debug info",
		"build/output.bin": "binary",
	}

	for path, content := range testStructure {
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0644) //nolint: gosec
		require.NoError(t, err)
	}

	err = os.WriteFile(".gitignore", []byte("*.env\nbuild/\n"), 0644) //nolint: gosec
	require.NoError(t, err)

	err = os.WriteFile("src/.gitignore", []byte("*.log\n"), 0644) //nolint: gosec
	require.NoError(t, err)

	ctx := context.Background()
	matchFn, err := GetIgnoreMatcherFn(ctx, nil, []string{".gitignore"}, false)
	require.NoError(t, err)

	archivePath := filepath.Join(tempDir, "test.tar.gz")
	tgz := *archiver.DefaultTarGz
	tgz.MatchFn = matchFn

	err = tgz.Archive([]string{"."}, archivePath)
	require.NoError(t, err)

	extractDir := filepath.Join(tempDir, "extracted")
	err = os.MkdirAll(extractDir, 0755)
	require.NoError(t, err)

	err = tgz.Unarchive(archivePath, extractDir)
	require.NoError(t, err)

	expectedIncluded := []string{
		"main.go",
		"config.yaml",
		"src/app.go",
		".gitignore",
		"src/.gitignore",
		"included.log",
	}

	expectedExcluded := []string{
		"secret.env",
		"src/debug.log",
		"build/output.bin",
	}

	for _, file := range expectedIncluded {
		fullPath := filepath.Join(extractDir, file)
		_, err := os.Stat(fullPath)
		assert.NoError(t, err, "Expected file %q to be included in archive", file)
	}

	for _, file := range expectedExcluded {
		fullPath := filepath.Join(extractDir, file)
		_, err := os.Stat(fullPath)
		assert.True(t, os.IsNotExist(err), "Expected file %q to be excluded from archive", file)
	}
}

func TestGetIgnoreMatcherFnAbsoluteProjectRootWithRelativePaths(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spacectl-abs-root-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Resolve symlinks (macOS /var -> /private/var) so Rel works correctly
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint: errcheck

	// Create a repo-like structure: tempDir is the "repo root" with a subdir
	subdir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	files := map[string]string{
		"root.txt":        "root file",
		"subdir/main.tf":  "resource {}",
		"subdir/vars.tf":  "variable {}",
		"other/other.txt": "other file",
	}
	for path, content := range files {
		full := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(full), 0755)
		require.NoError(t, err)
		err = os.WriteFile(full, []byte(content), 0644) //nolint: gosec
		require.NoError(t, err)
	}

	// Chdir to "repo root" to simulate MoveToRepositoryRoot
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use the absolute path as projectRoot (what MCP does)
	absSubdir := filepath.Join(tempDir, "subdir")

	// Convert absolute to relative (the fix) using CWD
	cwd, err := os.Getwd()
	require.NoError(t, err)
	relSubdir, err := filepath.Rel(cwd, absSubdir)
	require.NoError(t, err)

	ctx := context.Background()
	matchFn, err := GetIgnoreMatcherFn(ctx, &relSubdir, []string{".gitignore"}, false)
	require.NoError(t, err)

	// Archiver produces relative paths from "."
	testCases := map[string]bool{
		"subdir/main.tf":  true,
		"subdir/vars.tf":  true,
		"root.txt":        false, // outside project root
		"other/other.txt": false, // outside project root
	}

	for filePath, shouldInclude := range testCases {
		result := matchFn(filePath)
		assert.Equal(t, shouldInclude, result,
			"File %q: expected included=%v, got included=%v", filePath, shouldInclude, result)
	}
}

// TestGetIgnoreMatcherFnAbsoluteProjectRootWithArchiver is an end-to-end test
// verifying that an absolute projectRoot converted to relative works with the archiver.
func TestGetIgnoreMatcherFnAbsoluteProjectRootWithArchiver(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spacectl-abs-archiver-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd) //nolint: errcheck

	subdir := filepath.Join(tempDir, "subdir")
	files := map[string]string{
		"root.txt":       "root file",
		"subdir/main.tf": "resource {}",
		"subdir/vars.tf": "variable {}",
	}
	for path, content := range files {
		full := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(full), 0755)
		require.NoError(t, err)
		err = os.WriteFile(full, []byte(content), 0644) //nolint: gosec
		require.NoError(t, err)
	}

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Convert absolute to relative as the fix does
	relSubdir, err := filepath.Rel(tempDir, subdir)
	require.NoError(t, err)

	ctx := context.Background()
	matchFn, err := GetIgnoreMatcherFn(ctx, &relSubdir, []string{".gitignore"}, false)
	require.NoError(t, err)

	archivePath := filepath.Join(tempDir, "test.tar.gz")
	tgz := *archiver.DefaultTarGz
	tgz.MatchFn = matchFn

	err = tgz.Archive([]string{"."}, archivePath)
	require.NoError(t, err)

	extractDir := filepath.Join(tempDir, "extracted")
	err = os.MkdirAll(extractDir, 0755)
	require.NoError(t, err)

	err = tgz.Unarchive(archivePath, extractDir)
	require.NoError(t, err)

	// Only subdir files should be in the archive
	for _, file := range []string{"subdir/main.tf", "subdir/vars.tf"} {
		_, err := os.Stat(filepath.Join(extractDir, file))
		assert.NoError(t, err, "Expected file %q to be included in archive", file)
	}

	_, err = os.Stat(filepath.Join(extractDir, "root.txt"))
	assert.True(t, os.IsNotExist(err), "Expected root.txt to be excluded from archive")
}

func TestGetIgnoreMatcherFnWithProjectRoot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "spacectl-root-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	projectRoot := filepath.Join(tempDir, "project")
	err = os.MkdirAll(filepath.Join(projectRoot, "src"), 0755)
	require.NoError(t, err)

	outsideFile := filepath.Join(tempDir, "outside.txt")
	err = os.WriteFile(outsideFile, []byte("outside"), 0644) //nolint: gosec
	require.NoError(t, err)

	insideFile := filepath.Join(projectRoot, "inside.txt")
	err = os.WriteFile(insideFile, []byte("inside"), 0644) //nolint: gosec
	require.NoError(t, err)

	srcFile := filepath.Join(projectRoot, "src", "main.go")
	err = os.WriteFile(srcFile, []byte("package main"), 0644) //nolint: gosec
	require.NoError(t, err)

	ignoreFile := filepath.Join(projectRoot, ".gitignore")
	err = os.WriteFile(ignoreFile, []byte("*.log\n"), 0644) //nolint: gosec
	require.NoError(t, err)

	ctx := context.Background()
	matchFn, err := GetIgnoreMatcherFn(ctx, &projectRoot, []string{".gitignore"}, false)
	require.NoError(t, err)

	testCases := map[string]bool{
		"outside.txt":            false,
		tempDir + "/outside.txt": false,

		projectRoot + "/inside.txt":  true,
		projectRoot + "/src/main.go": true,
	}

	for filePath, shouldInclude := range testCases {
		result := matchFn(filePath)
		assert.Equal(t, shouldInclude, result,
			"File %q: expected included=%v, got included=%v", filePath, shouldInclude, result)
	}
}
