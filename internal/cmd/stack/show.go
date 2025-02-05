package stack

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

const (
	stackConfigVendorPulumi    = "StackConfigVendorPulumi"
	stackConfigVendorTerraform = "StackConfigVendorTerraform"
)

type stackConfigElement struct {
	ID        string `graphql:"id" json:"id,omitempty"`
	Checksum  string `graphql:"checksum" json:"checksum,omitempty"`
	Runtime   bool   `graphql:"runtime" json:"runtime"`
	Type      string `graphql:"type" json:"type,omitempty"`
	WriteOnly bool   `graphql:"writeOnly" json:"writeOnly"`
}

type showStackQuery struct {
	Stack *struct {
		ID               string `graphql:"id" json:"id,omitempty"`
		Administrative   bool   `graphql:"administrative" json:"administrative"`
		AttachedContexts []struct {
			ContextID string               `graphql:"contextId" json:"contextId,omitempty"`
			Name      string               `graphql:"contextName" json:"name,omitempty"`
			Priority  int                  `graphql:"priority" json:"priority,omitempty"`
			Config    []stackConfigElement `graphql:"config" json:"config,omitempty"`
		} `graphql:"attachedContexts"`
		AttachedPolicies []struct {
			PolicyID string `graphql:"policyId" json:"policyId,omitempty"`
			Name     string `graphql:"policyName" json:"name,omitempty"`
			Type     string `graphql:"policyType" json:"type,omitempty"`
		} `graphql:"attachedPolicies"`
		Autodeploy bool `graphql:"autodeploy" json:"autodeploy"`
		Autoretry  bool `graphql:"autoretry" json:"autoretry"`
		Blocker    struct {
			ID string `graphql:"id" json:"id,omitempty"`
		} `graphql:"blocker"`
		AfterApply          []string             `graphql:"afterApply" json:"afterApply,omitempty"`
		BeforeApply         []string             `graphql:"beforeApply" json:"beforeApply,omitempty"`
		AfterInit           []string             `graphql:"afterInit" json:"afterInit,omitempty"`
		BeforeInit          []string             `graphql:"beforeInit" json:"beforeInit,omitempty"`
		AfterPlan           []string             `graphql:"afterPlan" json:"afterPlan,omitempty"`
		BeforePlan          []string             `graphql:"beforePlan" json:"beforePlan,omitempty"`
		AfterPerform        []string             `graphql:"afterPerform" json:"afterPerform,omitempty"`
		BeforePerform       []string             `graphql:"beforePerform" json:"beforePerform,omitempty"`
		AfterDestroy        []string             `graphql:"afterDestroy" json:"afterDestroy,omitempty"`
		BeforeDestroy       []string             `graphql:"beforeDestroy" json:"beforeDestroy,omitempty"`
		Branch              string               `graphql:"branch" json:"branch,omitempty"`
		CanWrite            bool                 `graphql:"canWrite" json:"canWrite"`
		Config              []stackConfigElement `graphql:"config" json:"config,omitempty"`
		CreatedAt           int64                `graphql:"createdAt" json:"createdAt,omitempty"`
		Deleted             bool                 `graphql:"deleted" json:"deleted"`
		Deleting            bool                 `graphql:"deleting" json:"deleting"`
		Description         string               `graphql:"description" json:"description,omitempty"`
		Labels              []string             `graphql:"labels" json:"labels,omitempty"`
		LocalPreviewEnabled bool                 `graphql:"localPreviewEnabled" json:"localPreviewEnabled"`
		LockedBy            string               `graphql:"lockedBy" json:"lockedBy,omitempty"`
		ManagesStateFile    bool                 `graphql:"managesStateFile" json:"managesStateFile"`
		ModuleVersionsUsed  []struct {
			ID     string `graphql:"id" json:"id,omitempty"`
			Number string `graphql:"number" json:"number,omitempty"`
			Module struct {
				ID    string `graphql:"id" json:"id,omitempty"`
				Name  string `graphql:"name" json:"name,omitempty"`
				Owner string `graphql:"owner" json:"owner,omitempty"`
			} `graphql:"module" json:"module"`
		} `graphql:"moduleVersionsUsed" json:"moduleVersionsUsed,omitempty"`
		Name             string `graphql:"name" json:"name,omitempty"`
		Namespace        string `graphql:"namespace" json:"namespace,omitempty"`
		ProjectRoot      string `graphql:"projectRoot" json:"projectRoot,omitempty"`
		Provider         string `graphql:"provider" json:"provider,omitempty"`
		Repository       string `graphql:"repository" json:"repository,omitempty"`
		RunnerImage      string `graphql:"runnerImage" json:"runnerImage,omitempty"`
		Starred          bool   `graphql:"starred" json:"starred"`
		State            string `graphql:"state" json:"state,omitempty"`
		StateSetAt       int64  `graphql:"stateSetAt" json:"stateSetAt,omitempty"`
		TerraformVersion string `graphql:"terraformVersion" json:"terraformVersion,omitempty"`
		TrackedCommit    struct {
			AuthorLogin string `graphql:"authorLogin" json:"authorLogin,omitempty"`
			AuthorName  string `graphql:"authorName" json:"authorName,omitempty"`
			Hash        string `graphql:"hash" json:"hash,omitempty"`
			Message     string `graphql:"message" json:"message,omitempty"`
			Timestamp   int64  `graphql:"timestamp" json:"timestamp,omitempty"`
			URL         string `graphql:"url" json:"url,omitempty"`
		} `graphql:"trackedCommit" json:"trackedCommit,omitempty"`
		TrackedCommitSetBy string `graphql:"trackedCommitSetBy" json:"trackedCommitSetBy,omitempty"`
		VendorConfig       struct {
			Vendor string `graphql:"__typename"`
			Pulumi struct {
				LoginURL  string `graphql:"loginURL"`
				StackName string `graphql:"stackName"`
			} `graphql:"... on StackConfigVendorPulumi"`
			Terraform struct {
				Version   string `graphql:"version"`
				Workspace string `graphql:"workspace"`
			} `graphql:"... on StackConfigVendorTerraform"`
		} `graphql:"vendorConfig"`
		WorkerPool struct {
			ID   string `graphql:"id" json:"id,omitempty"`
			Name string `graphql:"name" json:"name,omitempty"`
		} `graphql:"workerPool" json:"workerPool,omitempty"`
	} `graphql:"stack(id: $stackId)" json:"stacks,omitempty"`
}

type showStackCommand struct{}

func (c *showStackCommand) showStack(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	var query showStackQuery
	variables := map[string]interface{}{
		"stackId": graphql.ID(stackID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return errors.Wrapf(err, "failed to query for stack ID %q", stackID)
	}

	if query.Stack == nil {
		return fmt.Errorf("stack ID %q not found", stackID)
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showStackTable(query)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(query.Stack)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *showStackCommand) showStackTable(query showStackQuery) error {
	c.outputStackNameSection(query)
	if err := c.outputVCSSettings(query); err != nil {
		return err
	}
	if err := c.outputBackendSettings(query); err != nil {
		return err
	}
	if err := c.outputBehaviorSettings(query); err != nil {
		return err
	}
	if err := c.outputContexts(query); err != nil {
		return err
	}
	if err := c.outputPolicies(query); err != nil {
		return err
	}
	if err := c.outputModuleVersionUsage(query); err != nil {
		return err
	}

	return nil
}

func (c *showStackCommand) outputStackNameSection(query showStackQuery) {
	pterm.DefaultSection.WithLevel(1).Print(query.Stack.Name)

	if len(query.Stack.Labels) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Labels")
		pterm.DefaultParagraph.Println(fmt.Sprintf("[%s]", strings.Join(query.Stack.Labels, "], [")))
	}

	if query.Stack.Description != "" {
		pterm.DefaultSection.WithLevel(2).Println("Description")
		pterm.DefaultParagraph.Println(query.Stack.Description)
	}
}

func (c *showStackCommand) outputVCSSettings(query showStackQuery) error {
	pterm.DefaultSection.WithLevel(2).Println("VCS Settings")
	tableData := [][]string{
		{"Provider", cmd.HumanizeVCSProvider(query.Stack.Provider)},
		{"Repository", query.Stack.Repository},
		{"Branch", query.Stack.Branch},
	}

	return cmd.OutputTable(tableData, false)
}

func (c *showStackCommand) outputBackendSettings(query showStackQuery) error {
	pterm.DefaultSection.WithLevel(2).Println("Backend")
	tableData := [][]string{
		{"Vendor", c.humanizeVendor(query.Stack.VendorConfig.Vendor)},
	}

	if query.Stack.VendorConfig.Vendor == stackConfigVendorPulumi {
		tableData = append(tableData, []string{"Login URL", query.Stack.VendorConfig.Pulumi.LoginURL})
		tableData = append(tableData, []string{"Stack name", query.Stack.VendorConfig.Pulumi.StackName})
	} else if query.Stack.VendorConfig.Vendor == stackConfigVendorTerraform {
		if !query.Stack.ManagesStateFile {
			tableData = append(tableData, []string{"Workspace", fmt.Sprint(query.Stack.VendorConfig.Terraform.Workspace)})
		}
		tableData = append(tableData, []string{"Version", stringWithDefault(query.Stack.VendorConfig.Terraform.Version, "latest")})
		tableData = append(tableData, []string{"Managed state", fmt.Sprint(query.Stack.ManagesStateFile)})
	}

	return cmd.OutputTable(tableData, false)
}

func (c *showStackCommand) outputBehaviorSettings(query showStackQuery) error {
	pterm.DefaultSection.WithLevel(2).Println("VCS Settings")

	workerPoolText := "Using shared public worker pool"
	if query.Stack.WorkerPool.ID != "" {
		workerPoolText = fmt.Sprintf("%s (%s)", query.Stack.WorkerPool.Name, query.Stack.WorkerPool.ID)
	}

	tableData := [][]string{
		{"Administrative", fmt.Sprint(query.Stack.Administrative)},
		{"Worker pool", workerPoolText},
		{"Autodeploy", fmt.Sprint(query.Stack.Autodeploy)},
		{"Autoretry", fmt.Sprint(query.Stack.Autoretry)},
		{"Local preview enabled", fmt.Sprint(query.Stack.LocalPreviewEnabled)},
		{"Project root", fmt.Sprint(query.Stack.ProjectRoot)},
		{"Runner image", stringWithDefault(query.Stack.RunnerImage, "default")},
	}

	if err := cmd.OutputTable(tableData, false); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.BeforeInit, "Before init scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.AfterInit, "After init scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.BeforePlan, "Before plan scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.AfterPlan, "After plan scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.BeforeApply, "Before apply scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.AfterApply, "After apply scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.BeforePerform, "Before perform scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.AfterPerform, "After perform scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.BeforeDestroy, "Before destroy scripts"); err != nil {
		return err
	}

	if err := c.outputScripts(query.Stack.AfterDestroy, "After destroy scripts"); err != nil {
		return err
	}

	return nil
}

func (c *showStackCommand) outputContexts(query showStackQuery) error {
	if len(query.Stack.AttachedContexts) == 0 {
		return nil
	}

	pterm.DefaultSection.WithLevel(2).Println("Attached contexts")

	sort.SliceStable(query.Stack.AttachedContexts, func(i, j int) bool {
		return query.Stack.AttachedContexts[i].Priority < query.Stack.AttachedContexts[j].Priority
	})

	tableData := [][]string{{"Priority", "Name", "ID"}}
	for _, context := range query.Stack.AttachedContexts {

		tableData = append(tableData, []string{
			fmt.Sprint(context.Priority),
			context.Name,
			context.ContextID,
		})
	}

	return cmd.OutputTable(tableData, true)
}

func (c *showStackCommand) outputPolicies(query showStackQuery) error {
	if len(query.Stack.AttachedPolicies) == 0 {
		return nil
	}

	pterm.DefaultSection.WithLevel(2).Println("Attached policies")

	tableData := [][]string{{"Name", "Type"}}
	for _, policy := range query.Stack.AttachedPolicies {

		tableData = append(tableData, []string{
			policy.Name,
			cmd.HumanizePolicyType(policy.Type),
		})
	}

	return cmd.OutputTable(tableData, true)
}

func (c *showStackCommand) outputScripts(scripts []string, title string) error {
	if len(scripts) > 0 {
		pterm.DefaultSection.WithLevel(3).Println(title)
		var items []pterm.BulletListItem
		for _, script := range scripts {
			items = append(items, pterm.BulletListItem{Level: 0, Text: script})
		}

		if err := pterm.DefaultBulletList.WithItems(items).Render(); err != nil {
			return err
		}
	}

	return nil
}

func (c *showStackCommand) outputModuleVersionUsage(query showStackQuery) error {
	if len(query.Stack.ModuleVersionsUsed) == 0 {
		return nil
	}

	sort.SliceStable(query.Stack.ModuleVersionsUsed, func(i, j int) bool {
		a := query.Stack.ModuleVersionsUsed[i]
		b := query.Stack.ModuleVersionsUsed[j]

		return a.Module.Owner < b.Module.Owner || (a.Module.Owner == b.Module.Owner && a.Module.Name < b.Module.Name)
	})

	pterm.DefaultSection.WithLevel(2).Println("Modules Used")
	tableData := [][]string{{"Owner", "Name", "Version"}}
	for _, version := range query.Stack.ModuleVersionsUsed {
		tableData = append(tableData, []string{
			version.Module.Owner,
			version.Module.Name,
			version.Number,
		})
	}

	return cmd.OutputTable(tableData, true)
}

func (c *showStackCommand) humanizeVendor(vendorConfigType string) string {
	switch vendorConfigType {
	case stackConfigVendorPulumi:
		return "Pulumi"
	case stackConfigVendorTerraform:
		return "Terraform"
	}

	return vendorConfigType
}

func stringWithDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}
