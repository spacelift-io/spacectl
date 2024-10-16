package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/cli/v2/git"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func openCommandInBrowser(cliCtx *cli.Context) error {
	if stackID := cliCtx.String(flagStackID.Name); stackID != "" {
		return browser.OpenURL(authenticated.Client.URL(
			"/stack/%s",
			stackID,
		))
	}

	ignoreSubdir := cliCtx.Bool(flagIgnoreSubdir.Name)
	getCurrentBranch := cliCtx.Bool(flagCurrentBranch.Name)
	count := cliCtx.Int(flagSearchCount.Name)

	var subdir *string
	if !ignoreSubdir {
		got, err := getGitRepositorySubdir()
		if err != nil {
			return err
		}
		subdir = &got
	}

	var branch *string
	if getCurrentBranch {
		got, err := getGitCurrentBranch()
		if err != nil {
			return err
		}
		branch = &got
	}

	name, err := getRepositoryName()
	if err != nil {
		return err
	}

	return findAndOpenStackInBrowser(cliCtx.Context, &stackSearchParams{
		count:          count,
		projectRoot:    subdir,
		repositoryName: name,
		branch:         branch,
	})
}

func findAndOpenStackInBrowser(ctx context.Context, p *stackSearchParams) error {
	got, err := findAndSelectStack(ctx, p, false)
	if errors.Is(err, errNoStackFound) {
		return errors.New("No stacks using the provided search parameters, maybe it's in a different subdir?")
	}

	return browser.OpenURL(authenticated.Client.URL(
		"/stack/%s",
		got.ID,
	))
}

// getRepositoryName calls a git command to return a url
// for current repository origin which it parses and returnes
// a name/repository combo. Example result: spacelift/onboarding
func getRepositoryName() (string, error) {
	// In future we could just parse this from .git/config
	// but it's not that simple with submodules, this is much easier
	// but requires `git` to be installed on users machine.
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	gitUrl := strings.TrimSpace(string(out))
	return cleanupRepositoryString(gitUrl)
}

func cleanupRepositoryString(s string) (string, error) {
	validGitUrl := git.IsURL(s)
	if !validGitUrl {
		return "", fmt.Errorf("unsupported repository string: %s", s)
	}

	gitURL, err := git.ParseURL(s)
	if err != nil {
		return "", fmt.Errorf("error parsing git url: %v", err)
	}
	path := strings.TrimSuffix(strings.TrimPrefix(gitURL.Path, "/"), ".git")
	return path, nil
}

func getGitCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", errors.New("result for current branch is empty")
	}

	return result, nil
}

// getGitRepositorySubdir will traverse the path back to .git
// and return the path it took to get there.
func getGitRepositorySubdir() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("couldn't get current working directory: %w", err)
	}

	root := current
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("couldn't stat .git directory: %w", err)
		}

		if newRoot := filepath.Dir(root); newRoot != root {
			root = newRoot
		} else {
			return "", fmt.Errorf("couldn't find .git directory in %s or any of its parents", current)
		}
	}

	pathWithoutRoot, err := filepath.Rel(root, current)
	if err != nil {
		return "", fmt.Errorf("couldn't get relative path: %w", err)
	}

	if pathWithoutRoot == "." {
		return "", nil
	}

	return filepath.ToSlash(pathWithoutRoot), nil
}

type stackSearchParams struct {
	count int

	repositoryName string

	projectRoot *string
	branch      *string
}

type searchStacksResult struct {
	Stacks   []stack
	PageInfo structs.PageInfo
}

func searchStacks(ctx context.Context, input structs.SearchInput) (searchStacksResult, error) {
	var query struct {
		SearchStacksOutput struct {
			Edges []struct {
				Node stack `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchStacks(input: $input)"`
	}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		map[string]interface{}{"input": input},
		graphql.WithHeader("Spacelift-GraphQL-Query", "StacksPage"),
	); err != nil {
		return searchStacksResult{}, errors.Wrap(err, "failed search for stacks")
	}

	stacks := make([]stack, 0)
	for _, q := range query.SearchStacksOutput.Edges {
		stacks = append(stacks, q.Node)
	}

	return searchStacksResult{
		Stacks:   stacks,
		PageInfo: query.SearchStacksOutput.PageInfo,
	}, nil
}
