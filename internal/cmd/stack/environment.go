package stack

import (
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
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

type configElement struct {
	ID        string     `graphql:"id" json:"id,omitempty"`
	Checksum  string     `graphql:"checksum" json:"checksum,omitempty"`
	CreatedAt int64      `graphql:"createdAt" json:"createdAt,omitempty"`
	Runtime   bool       `graphql:"runtime" json:"runtime,omitempty"`
	Type      ConfigType `graphql:"type" json:"type,omitempty"`
	Value     *string    `graphql:"value" json:"value,omitempty"`
	WriteOnly bool       `graphql:"writeOnly" json:"writeOnly,omitempty"`
	FileMode  *string    `graphql:"fileMode" json:"fileMode,omitempty"`
}

type listEnvElementOutput struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Value    *string `json:"value"`
	FileMode *string `json:"fileMode"`
	// Context specifies the name of the context.
	Context *string `json:"context"`
	// Runtime is not printed, it's just used to determine output formatting.
	Runtime bool `json:"runtime"`
	// WriteOnly is not printed, it's just used to determine output formatting.
	WriteOnly bool `json:"writeOnly"`
}

type runtimeConfig struct {
	Context *struct {
		ID          string `graphql:"id" json:"id,omitempty"`
		ContextName string `graphql:"contextName" json:"contextName,omitempty"`
	} `graphql:"context" json:"context"`
	Element configElement `graphql:"element" json:"element,omitempty"`
}

type listEnvQuery struct {
	Stack struct {
		RuntimeConfig []runtimeConfig `graphql:"runtimeConfig" json:"runtimeConfig"`
	} `graphql:"stack(id: $stack)" json:"stack"`
}

type listEnvCommand struct{}

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

func (e *listEnvCommand) listEnv(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	stackID := cliCtx.String(flagStackID.Name)

	var query listEnvQuery
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return err
	}

	var elements []listEnvElementOutput
	for _, config := range query.Stack.RuntimeConfig {
		var contextName *string
		if config.Context != nil {
			contextName = &config.Context.ContextName
		}
		if element, err := config.Element.toConfigElementOutput(contextName); err == nil {
			elements = append(elements, element)
		} else {
			return err
		}
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return e.showOutputsTable(elements)
	case cmd.OutputFormatJSON:
		return e.showOutputsJSON(elements)
	default:
		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func (e *listEnvCommand) showOutputsTable(outputs []listEnvElementOutput) error {
	tableData := [][]string{{"Name", "Type", "Value", "File Mode", "Context"}}
	for _, output := range outputs {
		var row []string

		row = append(row, output.Name)
		row = append(row, output.Type)

		var value string
		switch {
		case output.Runtime:
			value = "<computed>"
		case output.WriteOnly:
			value = "*****"
		case output.Type == string(fileTypeConfig):
			value = output.trimmedValue()
		case output.Value != nil:
			value = *output.Value
		default:
			// keep value as empty string
		}
		row = append(row, value)

		if output.FileMode != nil {
			row = append(row, *output.FileMode)
		} else {
			row = append(row, "")
		}
		if output.Context != nil {
			row = append(row, *output.Context)
		} else {
			row = append(row, "")
		}

		tableData = append(tableData, row)
	}
	return cmd.OutputTable(tableData, true)
}

func (e *listEnvCommand) showOutputsJSON(outputs []listEnvElementOutput) error {
	return cmd.OutputJSON(outputs)
}

func (e *configElement) toConfigElementOutput(contextName *string) (listEnvElementOutput, error) {
	var value = e.Value

	if e.Type == fileTypeConfig && e.Value != nil {
		result, err := base64.StdEncoding.DecodeString(*e.Value)

		if err != nil {
			message := fmt.Sprintf("failed to decode base64-encoded file with id %s", e.ID)
			return listEnvElementOutput{}, errors.Wrapf(err, message)
		}

		stringValue := string(result)
		value = &stringValue
	}

	return listEnvElementOutput{
		Name:      e.ID,
		Type:      string(e.Type),
		Value:     value,
		FileMode:  e.FileMode,
		Context:   contextName,
		Runtime:   e.Runtime,
		WriteOnly: e.WriteOnly,
	}, nil
}

func (o *listEnvElementOutput) trimmedValue() string {
	if o.Value == nil {
		return ""
	}

	lineBreaks := regexp.MustCompile("(\r?\n)|\r")
	valueNoNewlines := string(lineBreaks.ReplaceAll([]byte(*o.Value), []byte(" ")))

	maxOutputLength := 80
	ellipsis := "..."

	if len(valueNoNewlines) > maxOutputLength {
		return fmt.Sprintf("%s%s", valueNoNewlines[:(maxOutputLength-len(ellipsis))], ellipsis)
	}

	return valueNoNewlines
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
