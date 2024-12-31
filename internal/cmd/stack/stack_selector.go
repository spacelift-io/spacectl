package stack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

var (
	errNoStackFound = errors.New("no stack found")
)

const (
	envPromptSkipKey = "SPACECTL_SKIP_STACK_PROMPT"
)

// getStackID will try to retrieve a stack ID from multiple sources.
// It will do so in the following order:
// 1. Check the --id flag, if set, use that value.
// 2. Check the --run flag, if set, try to get the stack associated with the run.
// 2. Check the current directory to determine repository and subdirectory and search for a stack.
func getStackID(cliCtx *cli.Context) (string, error) {
	stack, err := getStack(cliCtx)
	if err != nil {
		return "", err
	}

	return stack.ID, nil
}

func getStack(cliCtx *cli.Context) (*stack, error) {
	if cliCtx.IsSet(flagStackID.Name) {
		stackID := cliCtx.String(flagStackID.Name)
		stack, err := stackGetByID(cliCtx.Context, stackID)
		if errors.Is(err, errNoStackFound) {
			return nil, fmt.Errorf("stack with id %q could not be found. Please check that the stack exists and that you have access to it. To list available stacks run: spacectl stack list", stackID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to check if stack exists: %w", err)
		}

		return stack, nil
	} else if cliCtx.IsSet(flagRun.Name) {
		runID := cliCtx.String(flagRun.Name)
		stack, err := stackGetByRunID(cliCtx.Context, runID)
		if errors.Is(err, errNoStackFound) {
			return nil, fmt.Errorf("run with id %q was not found. Please check that the run exists and that you have access to it. To list available stacks run: spacectl stack run list", runID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get stack by run id: %w", err)
		}

		return stack, nil
	}

	subdir, err := getGitRepositorySubdir()
	if err != nil {
		return nil, err
	}

	name, err := getRepositoryName()
	if err != nil {
		return nil, err
	}

	skip := os.Getenv(envPromptSkipKey) == "true"

	got, err := findAndSelectStack(cliCtx.Context, &stackSearchParams{
		count:          50,
		projectRoot:    &subdir,
		repositoryName: name,
	}, !skip)
	if err != nil {
		if errors.Is(err, errNoStackFound) {
			return nil, fmt.Errorf("%w: no --id flag was provided and stack could not be found by searching the current directory", err)
		}

		return nil, err
	}

	return got, nil
}

func stackGetByID(ctx context.Context, stackID string) (*stack, error) {
	var query struct {
		Stack struct {
			stack
		} `graphql:"stack(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": graphql.ID(stackID),
	}

	err := authenticated.Client.Query(ctx, &query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query GraphQL API when checking if a stack exists: %w", err)
	}

	if query.Stack.ID != stackID {
		return nil, errNoStackFound
	}

	return &query.Stack.stack, nil
}

func stackGetByRunID(ctx context.Context, runID string) (*stack, error) {
	var query struct {
		RunStack struct {
			stack
		} `graphql:"runStack(runId: $runId)"`
	}

	variables := map[string]interface{}{
		"runId": graphql.ID(runID),
	}

	err := authenticated.Client.Query(ctx, &query, variables)
	if err != nil {
		if err.Error() == "not found" {
			return nil, errNoStackFound
		}

		return nil, fmt.Errorf("failed to query GraphQL API when getting stack by run id: %w", err)
	}

	return &query.RunStack.stack, nil
}

func findAndSelectStack(ctx context.Context, p *stackSearchParams, forcePrompt bool) (*stack, error) {
	conditions := []structs.QueryPredicate{
		{
			Field: "repository",
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]string{p.repositoryName},
			},
		},
	}

	if p.projectRoot != nil && *p.projectRoot != "" {
		root := strings.TrimSuffix(*p.projectRoot, "/")
		conditions = append(conditions, structs.QueryPredicate{
			Field: "projectRoot",
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]string{root, root + "/"},
			},
		})
	}

	if p.branch != nil {
		conditions = append(conditions, structs.QueryPredicate{
			Field: "branch",
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]string{*p.branch},
			},
		})
	}

	input := structs.SearchInput{
		First:      internal.Ptr(p.count),
		Predicates: &conditions,
	}

	result, err := searchStacks(ctx, input)
	if err != nil {
		return nil, err
	}

	items := []string{}
	found := map[string]stack{}
	for _, s := range result.Stacks {
		items = append(items, s.Name)
		found[s.Name] = s
	}

	if len(found) == 0 {
		return nil, errNoStackFound
	}

	selected := found[items[0]]
	if len(items) > 1 || forcePrompt {
		if len(items) == p.count {
			fmt.Printf("Search results exceeded maximum capacity (%d) some stacks might be missing\n", p.count)
		}
		if len(items) == 1 && forcePrompt {
			fmt.Printf("Enable auto-selection by setting '%s=true'\n", envPromptSkipKey)
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
			return nil, err
		}

		selected = found[result]
	}

	return &selected, nil
}
