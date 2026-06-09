package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type deployCommand struct{}

func (c *deployCommand) deploy(ctx context.Context, cliCmd *cli.Command) error {
	blueprintID := cliCmd.String(flagRequiredBlueprintID.Name)

	b, found, err := getBlueprintByID(ctx, blueprintID)
	if err != nil {
		return errors.Wrapf(err, "failed to query for blueprint ID %q", blueprintID)
	}

	if !found {
		return fmt.Errorf("blueprint with ID %q not found", blueprintID)
	}

	var templateInputs []BlueprintStackCreateInputPair

	if filePath := cliCmd.String(flagInputFile.Name); filePath != "" {
		if templateInputs, err = inputsFromFile(filePath, b.Inputs); err != nil {
			return err
		}
	} else {
		if templateInputs, err = promptForInputs(b, err); err != nil {
			return err
		}
	}

	var mutation struct {
		BlueprintCreateStack struct {
			StackID string `graphql:"stackID"`
		} `graphql:"blueprintCreateStack(id: $id, input: $input)"`
	}

	err = authenticated.Client().Mutate(ctx, &mutation, map[string]any{
		"id": blueprintID, "input": BlueprintStackCreateInput{TemplateInputs: templateInputs},
	},
	)
	if err != nil {
		return fmt.Errorf("failed to deploy stack from the blueprint: %w", err)
	}

	url := authenticated.Client().URL("/stack/%s", mutation.BlueprintCreateStack.StackID)
	fmt.Printf("\nCreated stack: %q", url)

	return nil
}

func formatLabel(input blueprintInput) string {
	if input.Description != "" {
		return fmt.Sprintf("%s (%s) - %s", input.Name, input.ID, input.Description)
	}
	return fmt.Sprintf("%s (%s)", input.Name, input.ID)
}

func promptForInputs(b blueprint, err error) ([]BlueprintStackCreateInputPair, error) {
	templateInputs := make([]BlueprintStackCreateInputPair, 0, len(b.Inputs))

	for _, input := range b.Inputs {
		var value string
		switch strings.ToLower(input.Type) {
		case "", "short_text", "long_text":
			value, err = promptForTextInput(input)
			if err != nil {
				return nil, err
			}
		case "secret":
			value, err = promptForSecretInput(input)
			if err != nil {
				return nil, err
			}
		case "number":
			value, err = promptForIntegerInput(input)
			if err != nil {
				return nil, err
			}
		case "float":
			value, err = promptForFloatInput(input)
			if err != nil {
				return nil, err
			}
		case "boolean":
			value, err = promptForBooleanInput(input)
			if err != nil {
				return nil, err
			}
		case "select":
			value, err = promptForSelectInput(input, input.Options)
			if err != nil {
				return nil, err
			}
		}

		templateInputs = append(templateInputs, BlueprintStackCreateInputPair{
			ID:    input.ID,
			Value: value,
		})
	}
	return templateInputs, err
}

func promptForTextInput(input blueprintInput) (string, error) {
	prompt := promptui.Prompt{
		Label:   formatLabel(input),
		Default: input.Default,
	}
	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to read text input for %q: %w", input.Name, err)
	}

	return result, nil
}

func promptForSecretInput(input blueprintInput) (string, error) {
	prompt := promptui.Prompt{
		Label:   formatLabel(input),
		Default: input.Default,
		Mask:    '*',
	}
	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to read secret input for %q: %w", input.Name, err)
	}

	return result, nil
}

func promptForIntegerInput(input blueprintInput) (string, error) {
	prompt := promptui.Prompt{
		Label:   formatLabel(input),
		Default: input.Default,
		Validate: func(s string) error {
			_, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("input must be an integer")
			}

			return nil
		},
	}
	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to read integer input for %q: %w", input.Name, err)
	}

	return result, nil
}

func promptForFloatInput(input blueprintInput) (string, error) {
	prompt := promptui.Prompt{
		Label:   formatLabel(input),
		Default: input.Default,
		Validate: func(s string) error {
			_, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return fmt.Errorf("input must be a float")
			}

			return nil
		},
	}
	result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to read float input for %q: %w", input.Name, err)
	}

	return result, nil
}

func promptForSelectInput(input blueprintInput, options []string) (string, error) {
	cursorPosition := 0
	if input.Default != "" {
		cursorPosition = slices.Index(options, input.Default)
	}

	sel := promptui.Select{
		Label:     formatLabel(input),
		Items:     options,
		CursorPos: cursorPosition,
	}

	_, result, err := sel.Run()
	if err != nil {
		return "", fmt.Errorf("failed to read selected input for %q: %w", input.Name, err)
	}

	return result, nil
}

func promptForBooleanInput(input blueprintInput) (string, error) {
	if input.Default != "" {
		def, err := strconv.ParseBool(input.Default)
		if err != nil { // Silently ignore invalid defaults and prompt the user.
			input.Default = "" // Earlier validation should prevent this.
		} else {
			input.Default = strconv.FormatBool(def)
		}
	}

	return promptForSelectInput(input, []string{"true", "false"})
}

// inputsFromFile reads blueprint inputs from a JSON file. The file must be a
// JSON object mapping input IDs to values. All blueprint inputs must be present
// in the file and no extra keys are allowed. All errors are aggregated before
// returning so the user sees the full set of problems at once.
func inputsFromFile(filePath string, inputs []blueprintInput) ([]BlueprintStackCreateInputPair, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file %q: %w", filePath, err)
	}

	var fileInputs map[string]json.RawMessage
	if err = json.Unmarshal(data, &fileInputs); err != nil {
		return nil, fmt.Errorf("failed to parse input file %q as JSON object: %w", filePath, err)
	}

	blueprintInputsByID := make(map[string]blueprintInput, len(inputs))
	for _, input := range inputs {
		blueprintInputsByID[input.ID] = input
	}

	var errs []string

	for id := range fileInputs {
		if _, ok := blueprintInputsByID[id]; !ok {
			errs = append(errs, fmt.Sprintf("  - extra input %q is not defined in the blueprint", id))
		}
	}

	result := make([]BlueprintStackCreateInputPair, 0, len(inputs))
	for _, input := range inputs {
		raw, ok := fileInputs[input.ID]
		if !ok {
			if input.Default != "" { // If omitted and has a default, use it.
				raw = []byte(fmt.Sprintf("%q", input.Default))
			} else {
				errs = append(errs, fmt.Sprintf("  - missing required input %q (%s)", input.ID, input.Name))
				continue
			}
		}

		value, parseErr := parseFileInputValue(input, raw)
		if parseErr != nil {
			errs = append(errs, fmt.Sprintf("  - input %q (%s): %s", input.ID, input.Name, parseErr))
			continue
		}

		result = append(result, BlueprintStackCreateInputPair{ID: input.ID, Value: value})
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("input file %q validation failed:\n%s", filePath, strings.Join(errs, "\n"))
	}

	return result, nil
}

// parseFileInputValue converts a raw JSON value to the string representation
// expected by the blueprint API. Text and select types require a JSON string.
// Number, float, and boolean accept either their native JSON type or a JSON
// string that parses to the correct type (e.g. "42", "3.14", "true").
func parseFileInputValue(input blueprintInput, raw json.RawMessage) (string, error) {
	switch strings.ToLower(input.Type) {
	case "", "short_text", "long_text", "secret":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", fmt.Errorf("must be a string")
		}
		return s, nil

	case "number":
		// Accept a JSON integer directly, or a JSON string that holds an integer.
		// Use *int64 so JSON null yields nil (json.Unmarshal(null, &int64) is a silent no-op).
		var n *int64
		if err := json.Unmarshal(raw, &n); err == nil && n != nil {
			return strconv.FormatInt(*n, 10), nil
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				return strconv.FormatInt(v, 10), nil
			}
		}
		return "", fmt.Errorf("must be an integer")

	case "float":
		// Accept a JSON number directly, or a JSON string that holds a float.
		// Use *float64 so JSON null yields nil.
		var f *float64
		if err := json.Unmarshal(raw, &f); err == nil && f != nil {
			return strconv.FormatFloat(*f, 'f', -1, 64), nil
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				return strconv.FormatFloat(v, 'f', -1, 64), nil
			}
		}
		return "", fmt.Errorf("must be a float")

	case "boolean":
		// Accept a JSON boolean directly, or a JSON string that holds a boolean.
		// Use *bool so JSON null yields nil instead of the silent no-op that
		// json.Unmarshal(null, &bool) produces.
		var b *bool
		if err := json.Unmarshal(raw, &b); err == nil && b != nil {
			return strconv.FormatBool(*b), nil
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if v, err := strconv.ParseBool(s); err == nil {
				return strconv.FormatBool(v), nil
			}
		}
		return "", fmt.Errorf("must be a boolean")

	case "select":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", fmt.Errorf("must be a string")
		}
		if !slices.Contains(input.Options, s) {
			return "", fmt.Errorf("must be one of [%s], got %q", strings.Join(input.Options, ", "), s)
		}
		return s, nil

	default:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", fmt.Errorf("must be a string")
		}
		return s, nil
	}
}
