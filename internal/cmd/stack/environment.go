package stack

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// ConfigType is a type of configuration element.
type ConfigType string

const (
	fileTypeConfig   = ConfigType("FILE_MOUNT")
	envVarTypeConfig = ConfigType("ENVIRONMENT_VARIABLE")
)

// ConfigInput represents the input required to create or update a config
// element.
type ConfigInput struct {
	ID        graphql.ID      `json:"id"`
	Type      ConfigType      `json:"type"`
	Value     graphql.String  `json:"value"`
	WriteOnly graphql.Boolean `json:"writeOnly"`
}

func setVar(cliCtx *cli.Context) error {
	if nArgs := cliCtx.NArg(); nArgs != 2 {
		return fmt.Errorf("expected two arguments to `environment setenv` but got %d", nArgs)
	}

	envName := cliCtx.Args().Get(0)
	envValue := cliCtx.Args().Get(1)

	stackID := cliCtx.String(flagStackID.Name)

	var mutation struct {
		ConfigElement struct {
			ID        string `graphql:"id"`
			WriteOnly bool   `graphql:"writeOnly"`
			Value     string `graphql:"value"`
		} `graphql:"stackConfigAdd(stack: $stack, config: $config)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"config": ConfigInput{
			ID:        graphql.ID(envName),
			Type:      envVarTypeConfig,
			Value:     graphql.String(envValue),
			WriteOnly: graphql.Boolean(cliCtx.Bool(flagEnvironmentWriteOnly.Name)),
		},
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Environment variable (%s) has been set!\n", mutation.ConfigElement.ID)
	if mutation.ConfigElement.WriteOnly {
		fmt.Printf("Value: %s \n", strings.Repeat("*", len(envValue)))
	} else {
		fmt.Printf("Value: %s \n", mutation.ConfigElement.Value)
	}
	fmt.Printf("Write only: %t \n", mutation.ConfigElement.WriteOnly)

	return nil
}

func mountFile(cliCtx *cli.Context) error {
	nArgs := cliCtx.NArg()

	envName := cliCtx.Args().Get(0)
	stackID := cliCtx.String(flagStackID.Name)

	var fileContent []byte
	var err error

	switch nArgs {
	case 1:
		fmt.Println("Reading from STDIN...")

		if fileContent, err = ioutil.ReadAll(os.Stdin); err != nil {
			return fmt.Errorf("couldn't read from STDIN: %w", err)
		}
	case 2:
		filePath := cliCtx.Args().Get(1)

		if fileContent, err = os.ReadFile(filePath); err != nil {
			return fmt.Errorf("couldn't read file from %s: %w", filePath, err)
		}
	default:
		return fmt.Errorf("expected max two arguments to `environment mount` but got %d", nArgs)
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
			Type:      fileTypeConfig,
			Value:     graphql.String(base64.StdEncoding.EncodeToString(fileContent)),
			WriteOnly: graphql.Boolean(cliCtx.Bool(flagEnvironmentWriteOnly.Name)),
		},
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("File has been mounted to /mnt/workspace/%s\n", mutation.ConfigElement.ID)
	fmt.Printf("Write only: %t \n", mutation.ConfigElement.WriteOnly)

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

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	if mutation.ConfigElement == nil {
		return fmt.Errorf("environment (%s) doesn't exist", envName)
	}

	fmt.Printf("Environment (%s) has been deleted!", envName)

	return nil
}
