package stack

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
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
	ID        graphql.ID `json:"id"`
	Type      ConfigType `json:"type"`
	Value     string     `json:"value"`
	WriteOnly bool       `json:"writeOnly"`
}

type configElement struct {
	ID        string     `graphql:"id" json:"id,omitempty"`
	Checksum  string     `graphql:"checksum" json:"checksum,omitempty"`
	CreatedAt int64      `graphql:"createdAt" json:"createdAt,omitempty"`
	Runtime   bool       `graphql:"runtime" json:"runtime,omitempty"`
	Type      ConfigType `graphql:"type" json:"type,omitempty"`
	Value     *string    `graphql:"value" json:"value,omitempty"`
	WriteOnly bool       `graphql:"writeOnly" json:"writeOnly,omitempty"`
}

type listEnvElementOutput struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Value *string `json:"value"`

	// Context specifies the name of the context.
	Context        *string `json:"context"`
	IsAutoattached *bool   `json:"isAutoattached"`

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
		RuntimeConfig    []runtimeConfig `graphql:"runtimeConfig" json:"runtimeConfig"`
		AttachedContexts []struct {
			ContextID      string          `graphql:"contextId" json:"contextId,omitempty"`
			Name           string          `graphql:"contextName" json:"name,omitempty"`
			Priority       int             `graphql:"priority" json:"priority,omitempty"`
			IsAutoattached bool            `graphql:"isAutoattached" json:"isAutoattached"`
			Config         []configElement `graphql:"config" json:"config,omitempty"`
		} `graphql:"attachedContexts"`
	} `graphql:"stack(id: $stack)" json:"stack"`
}

type listEnvCommand struct{}

func setVar(cliCtx *cli.Context) error {
	if nArgs := cliCtx.NArg(); nArgs != 2 {
		return fmt.Errorf("expected two arguments to `environment setvar` but got %d", nArgs)
	}

	envName := cliCtx.Args().Get(0)
	envValue := cliCtx.Args().Get(1)

	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

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
			Value:     envValue,
			WriteOnly: cliCtx.Bool(flagEnvironmentWriteOnly.Name),
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

	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	var query listEnvQuery
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return err
	}

	var elements []listEnvElementOutput
	for _, config := range query.Stack.RuntimeConfig {
		config := config
		var contextName *string
		var isAutoAttached *bool
		if config.Context != nil {
			contextName = &config.Context.ContextName

			f := false
			isAutoAttached = &f
		}

		if element, err := config.Element.toConfigElementOutput(contextName, isAutoAttached); err == nil {
			elements = append(elements, element)
		} else {
			return err
		}
	}

	for _, spcCtx := range query.Stack.AttachedContexts {
		// If the context is not autoattached, we will get it with the whole config.
		// If it's autoattached, we have to specifically list and attach it.
		if !spcCtx.IsAutoattached {
			continue
		}

		for _, config := range spcCtx.Config {
			if element, err := config.toConfigElementOutput(&spcCtx.Name, &spcCtx.IsAutoattached); err == nil {
				elements = append(elements, element)
			} else {
				return err
			}
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
	tableData := [][]string{{"Name", "Type", "Value", "Context", "IsAutoattached"}}
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

		if output.Context != nil {
			row = append(row, *output.Context)
		} else {
			row = append(row, "")
		}

		if output.IsAutoattached != nil {
			row = append(row, fmt.Sprintf("%v", *output.IsAutoattached))
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

func (e *configElement) toConfigElementOutput(contextName *string, isAutoAttached *bool) (listEnvElementOutput, error) {
	var value = e.Value

	if e.Type == fileTypeConfig && e.Value != nil {
		result, err := base64.StdEncoding.DecodeString(*e.Value)

		if err != nil {
			message := fmt.Sprintf("failed to decode base64-encoded file with id %s", e.ID)
			return listEnvElementOutput{}, errors.Wrap(err, message)
		}

		stringValue := string(result)
		value = &stringValue
	}

	return listEnvElementOutput{
		Name:           e.ID,
		Type:           string(e.Type),
		Value:          value,
		Context:        contextName,
		IsAutoattached: isAutoAttached,
		Runtime:        e.Runtime,
		WriteOnly:      e.WriteOnly,
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
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	var fileContent []byte

	switch nArgs {
	case 1:
		fmt.Println("Reading from STDIN...")

		if fileContent, err = io.ReadAll(os.Stdin); err != nil {
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
			Value:     base64.StdEncoding.EncodeToString(fileContent),
			WriteOnly: cliCtx.Bool(flagEnvironmentWriteOnly.Name),
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
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

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
