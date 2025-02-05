package stack

import (
	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func resourcesList(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		if !errors.Is(err, errNoStackFound) {
			return err
		}

		return resourcesListAllStacks(cliCtx)
	}

	return resourcesListOneStack(cliCtx, stackID)
}

func resourcesListOneStack(cliCtx *cli.Context, id string) error {
	var query struct {
		Stack stackWithResources `graphql:"stack(id: $id)"`
	}

	variables := map[string]any{"id": graphql.ID(id)}
	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return errors.Wrap(err, "failed to query one stack")
	}

	return cmd.OutputJSON(query.Stack)
}

func resourcesListAllStacks(cliCtx *cli.Context) error {
	var query struct {
		Stacks []stackWithResources `graphql:"stacks" json:"stacks,omitempty"`
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "failed to query list of stacks")
	}

	return cmd.OutputJSON(query.Stacks)
}

type stackWithResources struct {
	ID     string   `graphql:"id" json:"id"`
	Labels []string `graphql:"labels" json:"labels"`
	Space  string   `graphql:"space" json:"space"`
	Name   string   `graphql:"name" json:"name"`

	Entities []managedEntity `graphql:"entities" json:"entities"`
}

type managedEntity struct {
	ID                 string       `graphql:"id" json:"id,omitempty"`
	Address            string       `graphql:"address" json:"address,omitempty"`
	Creator            *run         `graphql:"creator" json:"creator,omitempty"`
	Drifted            *int64       `graphql:"drifted" json:"drifted,omitempty"`
	Name               string       `graphql:"name" json:"name,omitempty"`
	Parent             string       `graphql:"parent" json:"parent,omitempty"`
	ThirdPartyMetadata *string      `graphql:"thirdPartyMetadata" json:"third_party_metadata,omitempty"`
	Type               string       `graphql:"type" json:"type,omitempty"`
	Updater            *run         `graphql:"updater" json:"updater,omitempty"`
	Vendor             entityVendor `graphql:"vendor" json:"vendor,omitempty"`
}

type run struct {
	ID string `graphql:"id" json:"id,omitempty"`
}

type entityVendor struct {
	EntityVendorAnsible struct {
		Ansible *ansibleEntity `graphql:"ansible" json:"ansible,omitempty"`
	} `graphql:"... on EntityVendorAnsible" json:"entity_vendor_ansible,omitempty"`
	EntityVendorCloudFormation struct {
		CloudFormation *cloudFormationEntity `graphql:"cloudFormation" json:"cloudFormation,omitempty"`
	} `graphql:"... on EntityVendorCloudFormation" json:"entity_vendor_cloud_formation,omitempty"`
	EntityVendorKubernetes struct {
		Kubernetes *kubernetesEntity `graphql:"kubernetes" json:"kubernetes,omitempty"`
	} `graphql:"... on EntityVendorKubernetes" json:"entity_vendor_kubernetes,omitempty"`
	EntityVendorPulumi struct {
		Pulumi *pulumiEntity `graphql:"pulumi" json:"pulumi,omitempty"`
	} `graphql:"... on EntityVendorPulumi" json:"entity_vendor_pulumi,omitempty"`
	EntityVendorTerraform struct {
		Terraform *terraformEntity `graphql:"terraform" json:"terraform,omitempty"`
	} `graphql:"... on EntityVendorTerraform" json:"entity_vendor_terraform,omitempty"`
}

type ansibleEntity struct {
	AnsibleResource struct {
		Data string `graphql:"data" json:"data,omitempty"`
	} `graphql:"... on AnsibleResource" json:"ansible_resource,omitempty"`
}

type cloudFormationEntity struct {
	CloudFormationResource struct {
		LogicalResourceID  string `graphql:"logicalResourceId" json:"logical_resource_id,omitempty"`
		PhysicalResourceID string `graphql:"physicalResourceId" json:"physical_resource_id,omitempty"`
		Template           string `graphql:"template" json:"template,omitempty"`
	} `graphql:"... on CloudFormationResource" json:"cloud_formation_resource,omitempty"`
	CloudFormationOutput struct {
		Description *string `graphql:"description" json:"description,omitempty"`
		Export      *string `graphql:"export" json:"export,omitempty"`
		Value       string  `graphql:"value" json:"value,omitempty"`
	} `graphql:"... on CloudFormationOutput" json:"cloud_formation_output,omitempty"`
}

type kubernetesEntity struct {
	KubernetesResource struct {
		Data string `graphql:"data" json:"data,omitempty"`
	} `graphql:"... on KubernetesResource" json:"kubernetes_resource,omitempty"`
	KubernetesRoot struct {
		Phantom bool `graphql:"phantom" json:"phantom,omitempty"`
	} `graphql:"... on KubernetesRoot" json:"kubernetes_root,omitempty"`
}

type pulumiEntity struct {
	PulumiOutput struct {
		Sensitive bool    `graphql:"sensitive" json:"sensitive,omitempty"`
		Value     *string `graphql:"value" json:"value,omitempty"`
		Hash      string  `graphql:"hash" json:"hash,omitempty"`
	} `graphql:"... on PulumiOutput" json:"pulumi_output,omitempty"`
	PulumiStack struct {
		Phantom bool `graphql:"phantom" json:"phantom,omitempty"`
	} `graphql:"... on PulumiStack" json:"pulumi_stack,omitempty"`
	PulumiResource struct {
		Urn      string `graphql:"urn" json:"urn,omitempty"`
		ID       string `graphql:"id" json:"id,omitempty"`
		Provider string `graphql:"provider" json:"provider,omitempty"`
		Parent   string `graphql:"parent" json:"parent,omitempty"`
		Outputs  string `graphql:"outputs" json:"outputs,omitempty"`
	} `graphql:"... on PulumiResource " json:"pulumi_resource,omitempty"`
}

type terraformEntity struct {
	TerraformResource struct {
		Address  string `graphql:"address" json:"address,omitempty"`
		Mode     string `graphql:"mode" json:"mode,omitempty"`
		Module   string `graphql:"module" json:"module,omitempty"`
		Provider string `graphql:"provider" json:"provider,omitempty"`
		Tainted  bool   `graphql:"tainted" json:"tainted,omitempty"`
		Values   string `graphql:"values" json:"values,omitempty"`
	} `graphql:"... on TerraformResource" json:"terraform_resource,omitempty"`
	TerraformModule struct {
		Phantom bool `graphql:"phantom" json:"phantom,omitempty"`
	} `graphql:"... on TerraformModule" json:"terraform_module,omitempty"`
	TerraformOutput struct {
		Sensitive bool    `graphql:"sensitive" json:"sensitive,omitempty"`
		Value     *string `graphql:"value" json:"value,omitempty"`
		Hash      string  `graphql:"hash" json:"hash,omitempty"`
	} `graphql:"... on TerraformOutput" json:"terraform_output,omitempty"`
}
