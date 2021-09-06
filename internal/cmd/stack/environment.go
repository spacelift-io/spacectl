package stack

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// ConfigType is a type of configuration element.
type ConfigType string

// ConfigInput represents the input required to create or update a config
// element.
type ConfigInput struct {
	ID        graphql.ID      `json:"id"`
	Type      ConfigType      `json:"type"`
	Value     graphql.String  `json:"value"`
	WriteOnly graphql.Boolean `json:"writeOnly"`
}

func setEnvironment(cliCtx *cli.Context) error {
	var value string
	var configType ConfigType

	if nArgs := cliCtx.NArg(); nArgs != 2 {
		return fmt.Errorf("expecting environment set as two arguments, got %d instead", nArgs)
	}

	envName := cliCtx.Args().Get(0)
	envValue := cliCtx.Args().Get(1)
	stackID := cliCtx.String(flagStackID.Name)

	if cliCtx.Bool(flagEnvironmentFile.Name) {
		fileContent, err := os.ReadFile(envValue)
		if err != nil {
			return err
		}

		configType = ConfigType("FILE_MOUNT")
		value = base64.StdEncoding.EncodeToString(fileContent)
	} else {
		configType = ConfigType("ENVIRONMENT_VARIABLE")
		value = envValue
	}

	var mutation struct {
		ConfigElement struct {
			ID        string     `graphql:"id"`
			WriteOnly bool       `graphql:"writeOnly"`
			Type      ConfigType `graphql:"type"`
		} `graphql:"stackConfigAdd(stack: $stack, config: $config)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"config": ConfigInput{
			ID:        graphql.ID(envName),
			Type:      configType,
			Value:     graphql.String(value),
			WriteOnly: graphql.Boolean(cliCtx.Bool(flagEnvironmentWriteOnly.Name)),
		},
	}

	if err := authenticated.Client.Mutate(context.Background(), &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Environment (%s) has been set!\n", mutation.ConfigElement.ID)
	fmt.Printf("Write only: %t \n", mutation.ConfigElement.WriteOnly)
	fmt.Printf("Type: %s \n", mutation.ConfigElement.Type)

	return nil
}

func deleteEnvironment(cliCtx *cli.Context) error {
	stackID := cliCtx.String(flagStackID.Name)

	if nArgs := cliCtx.NArg(); nArgs != 1 {
		return fmt.Errorf("expecting environment delete as only one agument, got %d instead", nArgs)
	}

	envName := cliCtx.Args().Get(0)

	var mutation struct {
		ConfigElement *struct {
			ID string
		} `graphql:"stackConfigDelete(stack: $stack, id: $id)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"id":    graphql.ID(envName),
	}

	if err := authenticated.Client.Mutate(context.Background(), &mutation, variables); err != nil {
		return err
	}

	if mutation.ConfigElement == nil {
		return fmt.Errorf("environment (%s) doesn't exist", envName)
	}

	fmt.Printf("Environment (%s) has been deleted!", envName)

	return nil
}
