package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

const rocketEmoji = "\U0001F680"

func runReplan(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}

	runID := cliCmd.String(flagRequiredRun.Name)

	var resources []string
	if cliCmd.Bool(flagInteractive.Name) {
		var err error
		resources, err = interactiveResourceSelection(ctx, stackID, runID)
		if err != nil {
			return err
		}
	} else {
		resources = cliCmd.StringSlice(flagResources.Name)
	}

	if len(resources) == 0 {
		return fmt.Errorf("no resources targeted for replanning: at least one resource must be specified")
	}

	var mutation struct {
		RunTargetedReplan struct {
			ID string `graphql:"id"`
		} `graphql:"runTargetedReplan(stack: $stack, run: $run, targets: $targets)"`
	}

	targets := make([]graphql.String, len(resources))
	for i, resource := range resources {
		targets[i] = graphql.String(resource)
	}

	variables := map[string]interface{}{
		"stack":   graphql.ID(stackID),
		"run":     graphql.ID(runID),
		"targets": targets,
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Run ID %q is being replanned\n", runID)
	fmt.Println("The live run can be visited at", authenticated.Client().URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.RunTargetedReplan.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := logs.NewExplorer(stackID, mutation.RunTargetedReplan.ID).RunFilteredLogs(ctx)
	if err != nil {
		return err
	}

	return terminal.Error()
}

func interactiveResourceSelection(ctx context.Context, stackID, runID string) ([]string, error) {
	resources, err := getRunChanges(ctx, stackID, runID)
	if err != nil {
		return nil, err
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   fmt.Sprintf("%s {{ .Address | cyan }} %s", rocketEmoji, rocketEmoji),
		Inactive: "  {{ .Address | cyan }}",
		Selected: fmt.Sprintf("%s {{ .Address cyan }} %s", rocketEmoji, rocketEmoji),
		Details: `
----------- Details ------------
{{ "Address:" | faint }}	{{ .Address }}
{{ "PreviousAddress:" | faint }}	{{ .PreviousAddress }}
{{ "Type:" | faint }}	{{ .Metadata.Type }}
`,
	}

	values := make([]runChangesResource, 0)
	selected := make([]string, 0)

	for _, r := range resources {
		values = append(values, r.Resources...)
	}

	for {
		prompt := promptui.Select{
			Label:             "Which resource should be added to the replan",
			Items:             values,
			Templates:         templates,
			Size:              20,
			StartInSearchMode: len(values) > 10,
			Searcher: func(input string, index int) bool {
				return strings.Contains(values[index].Address, input)
			},
		}

		index, _, err := prompt.Run()
		if err != nil {
			return nil, err
		}

		selected = append(selected, values[index].Address)
		values = append(values[:index], values[index+1:]...)

		if !shouldPickMore() || len(values) == 0 {
			break
		}
	}

	return selected, nil
}

func shouldPickMore() bool {
	prompt := promptui.Prompt{
		Label:     "Pick more",
		IsConfirm: true,
		Default:   "y",
	}

	result, _ := prompt.Run()

	return result == "y" || result == ""
}
