package workerpools

import (
	"context"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/draw"
	"github.com/spacelift-io/spacectl/internal/cmd/draw/data"
)

func watch(ctx context.Context, _ *cli.Command) error {
	got, err := findAndSelectWorkerPool(ctx)
	if err != nil {
		return err
	}

	wp := &data.WorkerPool{WokerPoolID: got}
	t, err := draw.NewTable(ctx, wp)
	if err != nil {
		return err
	}

	return t.DrawTable()
}

// findAndSelectWorkerPool finds all worker pools and lets the user select one.
//
// Returns the ID of the selected worker pool.
// If public worker pool is selected and empty string is returned.
func findAndSelectWorkerPool(ctx context.Context) (string, error) {
	var query listPoolsQuery
	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
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
