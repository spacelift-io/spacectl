package stack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var errNoStackFound = errors.New("no stack found")

// getStackID will try to retreive a stack ID from multiple sources.
// It will do so in the following order:
// 1. Check the --id flag, if set, use that value.
// 2. Check env variable SPACECTL_STACK_ID and use it if it's set to a non empty string.
// 3. Check the current directory to determine repository and subdirectory and search for a stack.
func getStackID(cliCtx *cli.Context) (string, error) {
	if cliCtx.IsSet(flagStackID.Name) {
		return cliCtx.String(flagStackID.Name), nil
	}

	if got := os.Getenv("SPACELIFT_CTL_STACK_ID"); got != "" {
		return got, nil
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
			return "", errors.New("no --id flag was provided and stack could not be found by searching current directory or env variables")
		}

		return "", err
	}

	return got, nil
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

	selected := items[0]
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

		selected = result
	}

	return selected, nil
}
