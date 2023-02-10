package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func openCommandInBrowser(cliCtx *cli.Context) error {
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

	return findAndSelect(cliCtx.Context, &stackSearchParams{
		count:          count,
		projectRoot:    subdir,
		repositoryName: name,
		branch:         branch,
	})
}

func findAndSelect(ctx context.Context, p *stackSearchParams) error {
	stacks, err := searchStacks(ctx, p)
	if err != nil {
		return err
	}

	items := []string{}
	found := map[string]string{}
	for _, stack := range stacks {
		items = append(items, stack.Name)
		found[stack.Name] = stack.ID
	}

	if len(found) == 0 {
		return errors.New("Didn't find stacks")
	}

	selected := items[0]
	if len(items) > 1 {
		if len(items) == p.count {
			fmt.Printf("Search results exceeded maximum capacity (%d) some stacks might be missing\n", p.count)
		}

		prompt := promptui.Select{
			Label:             fmt.Sprintf("Found %d stacks, select one", len(items)),
			Items:             items,
			Size:              10,
			StartInSearchMode: len(items) > 5,
			Searcher: func(input string, index int) bool {
				return strings.Contains(items[index], input)
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			return err
		}
		selected = result
	}

	return internal.OpenWebBrowser(authenticated.Client.URL(
		"/stack/%s",
		found[selected],
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

	result := strings.SplitN(string(out), ":", 2)
	if len(result) != 2 {
		return "", errors.New("could not parse result")
	}
	return strings.TrimSuffix(strings.TrimSpace(result[1]), ".git"), nil
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

// getGitRepositorySubdir will travese the path back to .git
// and return the path it took to get there.
func getGitRepositorySubdir() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("couldn't get current working directory: %w", err)
	}

	root := current
	for {
		if _, err := os.Stat(".git"); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("couldn't stat .git directory: %w", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("couldn't get current working directory: %w", err)
		}

		parent := filepath.Dir(cwd)
		if err := os.Chdir(parent); err != nil {
			return "", fmt.Errorf("couldn't set current working directory: %w", err)
		}
		root = parent
	}

	return strings.TrimPrefix(strings.ReplaceAll(current, root, ""), "/"), nil
}

type stackSearchParams struct {
	count int

	repositoryName string

	projectRoot *string
	branch      *string
}

func searchStacks(ctx context.Context, p *stackSearchParams) ([]stack, error) {
	var query struct {
		SearchStacksOutput struct {
			Edges []struct {
				Node stack `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchStacks(input: $input)"`
	}
	conditions := []structs.QueryPredicate{
		{
			Field: graphql.String("repository"),
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]graphql.String{graphql.String(p.repositoryName)},
			},
		},
	}

	if p.projectRoot != nil {
		conditions = append(conditions, structs.QueryPredicate{
			Field: graphql.String("projectRoot"),
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]graphql.String{graphql.String(*p.projectRoot)},
			},
		})
	}

	if p.branch != nil {
		conditions = append(conditions, structs.QueryPredicate{
			Field: graphql.String("branch"),
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]graphql.String{graphql.String(*p.branch)},
			},
		})
	}

	variables := map[string]interface{}{"input": structs.SearchInput{
		First:      graphql.NewInt(graphql.Int(p.count)),
		Predicates: &conditions,
	}}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		variables,
		graphql.WithHeader("Spacelift-GraphQL-Query", "StacksPage"),
	); err != nil {
		return nil, errors.Wrap(err, "failed search for stacks")
	}

	result := make([]stack, 0)
	for _, q := range query.SearchStacksOutput.Edges {
		result = append(result, q.Node)
	}

	return result, nil
}
