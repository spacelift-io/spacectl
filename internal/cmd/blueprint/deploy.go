package blueprint

import (
	"context"
	"fmt"
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

	templateInputs := make([]BlueprintStackCreateInputPair, 0, len(b.Inputs))

	for _, input := range b.Inputs {
		var value string
		switch strings.ToLower(input.Type) {
		case "", "short_text", "long_text":
			value, err = promptForTextInput(input)
			if err != nil {
				return err
			}
		case "secret":
			value, err = promptForSecretInput(input)
			if err != nil {
				return err
			}
		case "number":
			value, err = promptForIntegerInput(input)
			if err != nil {
				return err
			}
		case "float":
			value, err = promptForFloatInput(input)
			if err != nil {
				return err
			}
		case "boolean":
			value, err = promptForSelectInput(input, []string{"true", "false"})
			if err != nil {
				return err
			}
		case "select":
			value, err = promptForSelectInput(input, input.Options)
			if err != nil {
				return err
			}
		}

		templateInputs = append(templateInputs, BlueprintStackCreateInputPair{
			ID:    input.ID,
			Value: value,
		})
	}

	var mutation struct {
		BlueprintCreateStack struct {
			StackID string `graphql:"stackID"`
		} `graphql:"blueprintCreateStack(id: $id, input: $input)"`
	}

	err = authenticated.Client.Mutate(
		ctx,
		&mutation,
		map[string]any{
			"id": blueprintID,
			"input": BlueprintStackCreateInput{
				TemplateInputs: templateInputs,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to deploy stack from the blueprint: %w", err)
	}

	url := authenticated.Client.URL("/stack/%s", mutation.BlueprintCreateStack.StackID)
	fmt.Printf("\nCreated stack: %q", url)

	return nil
}

func formatLabel(input blueprintInput) string {
	if input.Description != "" {
		return fmt.Sprintf("%s (%s) - %s", input.Name, input.ID, input.Description)
	}
	return fmt.Sprintf("%s (%s)", input.Name, input.ID)
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
