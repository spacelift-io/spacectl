package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

var errNoStackFound = errors.New("no stack found")

type errStackWithIdNotFound struct {
	stackId string
}

func (e errStackWithIdNotFound) Error() string {
	return fmt.Sprintf("Stack with id %q could not be found. Please check that the stack exists and that you have access to it. To list available stacks run: spacectl stack list", e.stackId)
}

// getStackID will try to retreive a stack ID from multiple sources.
// It will do so in the following order:
// 1. Check the --id flag, if set, use that value.
// 2. Check the current directory to determine repository and subdirectory and search for a stack.
func getStackID(cliCtx *cli.Context) (string, error) {
	if cliCtx.IsSet(flagStackID.Name) {
		stackId := cliCtx.String(flagStackID.Name)
		err := stackExists(cliCtx.Context, stackId)
		if err != nil {
			return "", err
		}
		return stackId, nil
	}

	subdir, err := getGitRepositorySubdir()
	if err != nil {
		return "", err
	}

	name, err := getRepositoryName()
	if err != nil {
		return "", err
	}

	got, err := findAndSelectStack(cliCtx.Context, &stackSearchParams{
		count:          50,
		projectRoot:    &subdir,
		repositoryName: name,
	}, true)
	if err != nil {
		if errors.Is(err, errNoStackFound) {
			return "", fmt.Errorf("%w: no --id flag was provided and stack could not be found by searching the current directory", err)
		}

		return "", err
	}

	return got, nil
}

func stackExists(ctx context.Context, stackId string) error {
	var query struct {
		Stack struct {
			ID string `graphql:"id"`
		} `graphql:"stack(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": graphql.ID(stackId),
	}

	err := authenticated.Client.Query(ctx, &query, variables)
	if err != nil {
		return fmt.Errorf("failed to query GraphQL API: %w", err)
	}

	if query.Stack.ID == "" {
		return errStackWithIdNotFound{stackId: stackId}
	}

	return nil
}

func findAndSelectStack(ctx context.Context, p *stackSearchParams, forcePrompt bool) (string, error) {
	stacks, err := searchStacks(ctx, p)
	if err != nil {
		return "", err
	}

	items := []string{}
	found := map[string]string{}
	for _, stack := range stacks {
		items = append(items, stack.Name)
		found[stack.Name] = stack.ID
	}

	if len(found) == 0 {
		return "", errNoStackFound
	}

	selected := found[items[0]]
	if len(items) > 1 || forcePrompt {
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
			return "", err
		}

		selected = found[result]
	}

	return selected, nil
}
