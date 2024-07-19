package module

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func listModules() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return listModulesTable(cliCtx)
		case cmd.OutputFormatJSON:
			return listModulesJSON(cliCtx.Context)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listModulesJSON(ctx context.Context) error {
	var query struct {
		Modules []module `graphql:"modules" json:"modules,omitempty"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "failed to query list of modules")
	}
	return cmd.OutputJSON(query.Modules)
}

func listModulesTable(ctx *cli.Context) error {
	var query struct {
		Modules []struct {
			ID      string `graphql:"id" json:"id,omitempty"`
			Name    string `graphql:"name"`
			Current struct {
				ID     string `graphql:"id"`
				Number string `graphql:"number"`
				State  string `graphql:"state"`
				Yanked bool   `graphql:"yanked"`
			} `graphql:"current"`
		} `graphql:"modules"`
	}

	if err := authenticated.Client.Query(ctx.Context, &query, map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "failed to query list of modules")
	}

	sort.SliceStable(query.Modules, func(i, j int) bool {
		return strings.Compare(strings.ToLower(query.Modules[i].Name), strings.ToLower(query.Modules[j].Name)) < 0
	})

	columns := []string{"Name", "ID", "Current Version", "Number", "State", "Yanked"}

	tableData := [][]string{columns}
	for _, module := range query.Modules {
		row := []string{
			module.Name,
			module.ID,
			module.Current.ID,
			module.Current.Number,
			module.Current.State,
			fmt.Sprintf("%t", module.Current.Yanked),
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

type module struct {
	ID                  string   `json:"id" graphql:"id"`
	Administrative      bool     `json:"administrative" graphql:"administrative"`
	APIHost             string   `json:"apiHost" graphql:"apiHost"`
	Branch              string   `json:"branch" graphql:"branch"`
	Description         string   `json:"description" graphql:"description"`
	CanWrite            bool     `json:"canWrite" graphql:"canWrite"`
	IsDisabled          bool     `json:"isDisabled" graphql:"isDisabled"`
	CreatedAt           int      `json:"createdAt" graphql:"createdAt"`
	Labels              []string `json:"labels" graphql:"labels"`
	Name                string   `json:"name" graphql:"name"`
	Namespace           string   `json:"namespace" graphql:"namespace"`
	ProjectRoot         string   `json:"projectRoot" graphql:"projectRoot"`
	Provider            string   `json:"provider" graphql:"provider"`
	Repository          string   `json:"repository" graphql:"repository"`
	TerraformProvider   string   `json:"terraformProvider" graphql:"terraformProvider"`
	LocalPreviewEnabled bool     `json:"localPreviewEnabled,omitempty" graphql:"localPreviewEnabled"`
	Current             struct {
		ID     string `json:"id" graphql:"id"`
		Number string `json:"number" graphql:"number"`
		State  string `json:"state" graphql:"state"`
		Yanked bool   `json:"yanked" graphql:"yanked"`
	} `json:"current" graphql:"current"`
	SpaceDetails struct {
		ID          string `json:"id" graphql:"id"`
		Name        string `json:"name" graphql:"name"`
		AccessLevel string `json:"accessLevel" graphql:"accessLevel"`
	} `json:"spaceDetails" graphql:"spaceDetails"`
	WorkerPool struct {
		ID   string `graphql:"id" json:"id,omitempty"`
		Name string `graphql:"name" json:"name,omitempty"`
	} `graphql:"workerPool" json:"workerPool,omitempty"`
	Starred bool `json:"starred" graphql:"starred"`
}
