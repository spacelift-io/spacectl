package stack

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func dependenciesOn(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	got, err := dependenciesListOneStack(cliCtx)
	if err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return cmd.OutputTable(got.dependsOnTableData(), true)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(got.DependsOn)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func dependenciesOff(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	got, err := dependenciesListOneStack(cliCtx)
	if err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return cmd.OutputTable(got.dependedOnByTableData(), true)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(got.IsDependedOnBy)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func dependenciesListOneStack(cliCtx *cli.Context) (*stackWithDependencies, error) {
	id, err := getStackID(cliCtx)
	if err != nil {
		return nil, err
	}

	var query struct {
		Stack stackWithDependencies `graphql:"stack(id: $id)"`
	}

	variables := map[string]any{"id": graphql.ID(id)}
	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
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
