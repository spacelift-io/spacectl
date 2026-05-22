package stack

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const (
	stackConfigVendorAnsible        = "StackConfigVendorAnsible"
	stackConfigVendorCloudFormation = "StackConfigVendorCloudFormation"
	stackConfigVendorKubernetes     = "StackConfigVendorKubernetes"
	stackConfigVendorOpenTofu       = "StackConfigVendorOpenTofu"
	stackConfigVendorPulumi         = "StackConfigVendorPulumi"
	stackConfigVendorTerraform      = "StackConfigVendorTerraform"
	stackConfigVendorTerragrunt     = "StackConfigVendorTerragrunt"
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
		Name                   string   `graphql:"name" json:"name,omitempty"`
		Namespace              string   `graphql:"namespace" json:"namespace,omitempty"`
		ProjectRoot            string   `graphql:"projectRoot" json:"projectRoot,omitempty"`
		AdditionalProjectGlobs []string `graphql:"additionalProjectGlobs" json:"additionalProjectGlobs,omitempty"`
		Provider               string   `graphql:"provider" json:"provider,omitempty"`
		Repository             string   `graphql:"repository" json:"repository,omitempty"`
		RunnerImage            string   `graphql:"runnerImage" json:"runnerImage,omitempty"`
		Starred                bool     `graphql:"starred" json:"starred"`
		State                  string   `graphql:"state" json:"state,omitempty"`
		StateSetAt             int64    `graphql:"stateSetAt" json:"stateSetAt,omitempty"`
		TerraformVersion       string   `graphql:"terraformVersion" json:"terraformVersion,omitempty"`
		TrackedCommit          struct {
			AuthorLogin string `graphql:"authorLogin" json:"authorLogin,omitempty"`
			AuthorName  string `graphql:"authorName" json:"authorName,omitempty"`
			Hash        string `graphql:"hash" json:"hash,omitempty"`
			Message     string `graphql:"message" json:"message,omitempty"`
			Timestamp   int64  `graphql:"timestamp" json:"timestamp,omitempty"`
			URL         string `graphql:"url" json:"url,omitempty"`
		} `graphql:"trackedCommit" json:"trackedCommit"`
		TrackedCommitSetBy string `graphql:"trackedCommitSetBy" json:"trackedCommitSetBy,omitempty"`
		VendorConfig       struct {
			Vendor  string `graphql:"__typename"`
			Ansible struct {
				Playbook string `graphql:"playbook"`
			} `graphql:"... on StackConfigVendorAnsible"`
			CloudFormation struct {
				EntryTemplateFile string `graphql:"entryTemplateFile"`
				TemplateBucket    string `graphql:"templateBucket"`
				StackName         string `graphql:"stackName"`
				Region            string `graphql:"region"`
			} `graphql:"... on StackConfigVendorCloudFormation"`
			Kubernetes struct {
				Namespace      string `graphql:"namespace"`
				KubectlVersion string `graphql:"kubectlVersion"`
				WorkflowTool   string `graphql:"kubernetesWorkflowTool"`
			} `graphql:"... on StackConfigVendorKubernetes"`
			OpenTofu struct {
				Version                    string `graphql:"version"`
				Workspace                  string `graphql:"workspace"`
				UseSmartSanitization       bool   `graphql:"useSmartSanitization"`
				ExternalStateAccessEnabled bool   `graphql:"externalStateAccessEnabled"`
				Concise                    bool   `graphql:"concise"`
				WorkflowTool               string `graphql:"openTofuWorkflowTool: workflowTool"`
			} `graphql:"... on StackConfigVendorOpenTofu"`
			Pulumi struct {
				LoginURL  string `graphql:"loginURL"`
				StackName string `graphql:"stackName"`
			} `graphql:"... on StackConfigVendorPulumi"`
			Terraform struct {
				Version                    string `graphql:"version"`
				Workspace                  string `graphql:"workspace"`
				UseSmartSanitization       bool   `graphql:"useSmartSanitization"`
				ExternalStateAccessEnabled bool   `graphql:"externalStateAccessEnabled"`
				WorkflowTool               string `graphql:"terraformWorkflowTool: workflowTool"`
			} `graphql:"... on StackConfigVendorTerraform"`
			Terragrunt struct {
				TerraformVersion     string `graphql:"terraformVersion"`
				TerragruntVersion    string `graphql:"terragruntVersion"`
				UseRunAll            bool   `graphql:"useRunAll"`
				UseSmartSanitization bool   `graphql:"useSmartSanitization"`
				Tool                 string `graphql:"tool"`
				UseStateManagement   bool   `graphql:"useStateManagement"`
			} `graphql:"... on StackConfigVendorTerragrunt"`
		} `graphql:"vendorConfig"`
		WorkerPool struct {
			ID   string `graphql:"id" json:"id,omitempty"`
			Name string `graphql:"name" json:"name,omitempty"`
		} `graphql:"workerPool" json:"workerPool"`
	} `graphql:"stack(id: $stackId)" json:"stacks,omitempty"`
}

type showStackCommand struct{}

func (c *showStackCommand) showStack(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}

	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	var query showStackQuery
	variables := map[string]any{
		"stackId": graphql.ID(stackID),
	}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
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
	vc := query.Stack.VendorConfig
	tableData := [][]string{
		{"Vendor", humanizeVendor(vc.Vendor)},
	}

	switch vc.Vendor {
	case stackConfigVendorAnsible:
		tableData = append(tableData, []string{"Playbook", vc.Ansible.Playbook})
	case stackConfigVendorCloudFormation:
		tableData = append(tableData, []string{"Stack name", vc.CloudFormation.StackName})
		tableData = append(tableData, []string{"Region", vc.CloudFormation.Region})
		tableData = append(tableData, []string{"Entry template file", vc.CloudFormation.EntryTemplateFile})
		tableData = append(tableData, []string{"Template bucket", vc.CloudFormation.TemplateBucket})
	case stackConfigVendorKubernetes:
		tableData = append(tableData, []string{"Namespace", vc.Kubernetes.Namespace})
		tableData = append(tableData, []string{"Kubectl version", stringWithDefault(vc.Kubernetes.KubectlVersion, "latest")})
		tableData = append(tableData, []string{"Workflow tool", humanizeWorkflowTool(vc.Kubernetes.WorkflowTool)})
	case stackConfigVendorOpenTofu:
		if !query.Stack.ManagesStateFile {
			tableData = append(tableData, []string{"Workspace", vc.OpenTofu.Workspace})
		}
		tableData = append(tableData, []string{"Version", stringWithDefault(vc.OpenTofu.Version, "latest")})
		tableData = append(tableData, []string{"Workflow tool", humanizeWorkflowTool(vc.OpenTofu.WorkflowTool)})
		tableData = append(tableData, []string{"Managed state", fmt.Sprint(query.Stack.ManagesStateFile)})
		tableData = append(tableData, []string{"Smart sanitization", fmt.Sprint(vc.OpenTofu.UseSmartSanitization)})
		tableData = append(tableData, []string{"External state access", fmt.Sprint(vc.OpenTofu.ExternalStateAccessEnabled)})
		tableData = append(tableData, []string{"Concise", fmt.Sprint(vc.OpenTofu.Concise)})
	case stackConfigVendorPulumi:
		tableData = append(tableData, []string{"Login URL", vc.Pulumi.LoginURL})
		tableData = append(tableData, []string{"Stack name", vc.Pulumi.StackName})
	case stackConfigVendorTerraform:
		if !query.Stack.ManagesStateFile {
			tableData = append(tableData, []string{"Workspace", vc.Terraform.Workspace})
		}
		tableData = append(tableData, []string{"Version", stringWithDefault(vc.Terraform.Version, "latest")})
		tableData = append(tableData, []string{"Workflow tool", humanizeWorkflowTool(vc.Terraform.WorkflowTool)})
		tableData = append(tableData, []string{"Managed state", fmt.Sprint(query.Stack.ManagesStateFile)})
		tableData = append(tableData, []string{"Smart sanitization", fmt.Sprint(vc.Terraform.UseSmartSanitization)})
		tableData = append(tableData, []string{"External state access", fmt.Sprint(vc.Terraform.ExternalStateAccessEnabled)})
	case stackConfigVendorTerragrunt:
		tableData = append(tableData, []string{"Terragrunt version", stringWithDefault(vc.Terragrunt.TerragruntVersion, "latest")})
		tableData = append(tableData, []string{"Terraform version", stringWithDefault(vc.Terragrunt.TerraformVersion, "latest")})
		tableData = append(tableData, []string{"Tool", humanizeWorkflowTool(vc.Terragrunt.Tool)})
		tableData = append(tableData, []string{"Run-all", fmt.Sprint(vc.Terragrunt.UseRunAll)})
		tableData = append(tableData, []string{"State management", fmt.Sprint(vc.Terragrunt.UseStateManagement)})
		tableData = append(tableData, []string{"Smart sanitization", fmt.Sprint(vc.Terragrunt.UseSmartSanitization)})
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
		{"Additional project globs", strings.Join(query.Stack.AdditionalProjectGlobs, ", ")},
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

func humanizeVendor(vendorConfigType string) string {
	switch vendorConfigType {
	case stackConfigVendorAnsible:
		return "Ansible"
	case stackConfigVendorCloudFormation:
		return "CloudFormation"
	case stackConfigVendorKubernetes:
		return "Kubernetes"
	case stackConfigVendorOpenTofu:
		return "OpenTofu"
	case stackConfigVendorPulumi:
		return "Pulumi"
	case stackConfigVendorTerraform:
		return "Terraform"
	case stackConfigVendorTerragrunt:
		return "Terragrunt"
	}

	return vendorConfigType
}

// humanizeWorkflowTool maps backend workflow-tool enum values
// (TerraformWorkflowTool, OpenTofuWorkflowTool, KubernetesWorkflowTool, TerragruntTool)
// to readable names. The enums share value names so a single mapping is enough.
func humanizeWorkflowTool(tool string) string {
	switch tool {
	case "TERRAFORM_FOSS":
		return "Terraform"
	case "OPEN_TOFU", "OPENTOFU":
		return "OpenTofu"
	case "KUBERNETES":
		return "kubectl"
	case "MANUALLY_PROVISIONED":
		return "Manually provisioned"
	case "CUSTOM":
		return "Custom"
	}

	return tool
}

func stringWithDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}
