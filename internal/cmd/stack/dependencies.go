package stack

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func dependenciesOn(ctx context.Context, cmd *cli.Command) error {
	outputFormat, err := internalCmd.GetOutputFormat(cmd)
	if err != nil {
		return err
	}

	got, err := dependenciesListOneStack(ctx, cmd)
	if err != nil {
		return err
	}

	switch outputFormat {
	case internalCmd.OutputFormatTable:
		return internalCmd.OutputTable(got.dependsOnTableData(), true)
	case internalCmd.OutputFormatJSON:
		return internalCmd.OutputJSON(got.DependsOn)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func dependenciesOff(ctx context.Context, cmd *cli.Command) error {
	outputFormat, err := internalCmd.GetOutputFormat(cmd)
	if err != nil {
		return err
	}

	got, err := dependenciesListOneStack(ctx, cmd)
	if err != nil {
		return err
	}

	switch outputFormat {
	case internalCmd.OutputFormatTable:
		return internalCmd.OutputTable(got.dependedOnByTableData(), true)
	case internalCmd.OutputFormatJSON:
		return internalCmd.OutputJSON(got.IsDependedOnBy)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func dependenciesListOneStack(ctx context.Context, cmd *cli.Command) (*stackWithDependencies, error) {
	id, err := getStackID(ctx, cmd)
	if err != nil {
		return nil, err
	}

	var query struct {
		Stack stackWithDependencies `graphql:"stack(id: $id)"`
	}

	variables := map[string]any{"id": graphql.ID(id)}
	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to query one stack")
	}

	return &query.Stack, nil
}

type stackWithDependencies struct {
	ID     string   `graphql:"id" json:"id"`
	Labels []string `graphql:"labels" json:"labels"`
	Space  string   `graphql:"space" json:"space"`
	Name   string   `graphql:"name" json:"name"`

	DependsOn      []stackDependency `graphql:"dependsOn" json:"dependsOn"`
	IsDependedOnBy []stackDependency `graphql:"isDependedOnBy" json:"isDependedOnBy"`
}

type stackDependency struct {
	Stack struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"stack" json:"stack"`

	DependsOnStack struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"dependsOnStack" json:"dependsOnStack"`
}

func (s *stackWithDependencies) dependsOnTableData() [][]string {
	columns := []string{"Name", "ID"}

	tableData := [][]string{columns}
	for _, dependency := range s.DependsOn {
		tableData = append(tableData, []string{dependency.DependsOnStack.Name, dependency.DependsOnStack.ID})
	}

	return tableData
}

func (s *stackWithDependencies) dependedOnByTableData() [][]string {
	columns := []string{"Name", "ID"}

	tableData := [][]string{columns}
	for _, dependency := range s.IsDependedOnBy {
		tableData = append(tableData, []string{dependency.Stack.Name, dependency.Stack.ID})
	}

	return tableData
}
