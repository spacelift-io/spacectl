package workerpools

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/draw"
	"github.com/spacelift-io/spacectl/internal/cmd/draw/data"
)

func watch(cliCtx *cli.Context) error {
	got, err := findAndSelectWorkerPool(cliCtx)
	if err != nil {
		return err
	}

	wp := &data.WorkerPool{WokerPoolID: got}
	t, err := draw.NewTable(cliCtx.Context, wp)
	if err != nil {
		return err
	}

	return t.DrawTable()
}

// findAndSelectWorkerPool finds all worker pools and lets the user select one.
//
// Returns the ID of the selected worker pool.
// If public worker pool is selected and empty string is returned.
func findAndSelectWorkerPool(cliCtx *cli.Context) (string, error) {
	var query listPoolsQuery
	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{}); err != nil {
		return "", err
	}

	items := []string{"Public worker pool"}
	found := map[string]string{
		"Public worker pool": "",
	}
	for _, p := range query.Pools {
		items = append(items, p.Name)
		found[p.Name] = p.ID
	}

	prompt := promptui.Select{
		Label:             fmt.Sprintf("Found %d worker pools, select one", len(items)),
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

	return found[result], nil
}
