package blueprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempInputFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inputs.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestInputsFromFile_missingFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "nonexistent.json")
	_, err := inputsFromFile(filePath, []blueprintInput{{ID: "x", Name: "X", Type: "short_text"}})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read input file")
}

func TestInputsFromFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		inputs      []blueprintInput
		wantPairs   []BlueprintStackCreateInputPair
	}{
		{
			name:        "empty blueprint and empty file",
			fileContent: `{}`,
			inputs:      []blueprintInput{},
			wantPairs:   []BlueprintStackCreateInputPair{},
		},
		{
			name:        "short_text",
			fileContent: `{"env": "production"}`,
			inputs:      []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "env", Value: "production"}},
		},
		{
			name:        "long_text",
			fileContent: `{"desc": "a long description"}`,
			inputs:      []blueprintInput{{ID: "desc", Name: "Description", Type: "long_text"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "desc", Value: "a long description"}},
		},
		{
			name:        "secret",
			fileContent: `{"token": "s3cr3t"}`,
			inputs:      []blueprintInput{{ID: "token", Name: "Token", Type: "secret"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "token", Value: "s3cr3t"}},
		},
		{
			name:        "empty type treated as string",
			fileContent: `{"key": "val"}`,
			inputs:      []blueprintInput{{ID: "key", Name: "Key", Type: ""}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "key", Value: "val"}},
		},
		{
			name:        "unknown type treated as string",
			fileContent: `{"key": "val"}`,
			inputs:      []blueprintInput{{ID: "key", Name: "Key", Type: "custom_widget"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "key", Value: "val"}},
		},
		{
			name:        "type matching is case-insensitive",
			fileContent: `{"flag": true}`,
			inputs:      []blueprintInput{{ID: "flag", Name: "Flag", Type: "BOOLEAN"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "flag", Value: "true"}},
		},
		{
			name:        "empty string value",
			fileContent: `{"env": ""}`,
			inputs:      []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "env", Value: ""}},
		},
		{
			name:        "number positive integer",
			fileContent: `{"count": 42}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "count", Value: "42"}},
		},
		{
			name:        "number zero",
			fileContent: `{"count": 0}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "count", Value: "0"}},
		},
		{
			name:        "number negative integer",
			fileContent: `{"count": -7}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "count", Value: "-7"}},
		},
		{
			name:        "number as string",
			fileContent: `{"count": "42"}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "count", Value: "42"}},
		},
		{
			name:        "number negative as string",
			fileContent: `{"count": "-7"}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "count", Value: "-7"}},
		},
		{
			name:        "float with decimal part",
			fileContent: `{"ratio": 3.14}`,
			inputs:      []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "ratio", Value: "3.14"}},
		},
		{
			name:        "float whole number",
			fileContent: `{"ratio": 2}`,
			inputs:      []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "ratio", Value: "2"}},
		},
		{
			name:        "float negative",
			fileContent: `{"ratio": -1.5}`,
			inputs:      []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "ratio", Value: "-1.5"}},
		},
		{
			name:        "float as string",
			fileContent: `{"ratio": "3.14"}`,
			inputs:      []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "ratio", Value: "3.14"}},
		},
		{
			name:        "float whole number as string",
			fileContent: `{"ratio": "2"}`,
			inputs:      []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "ratio", Value: "2"}},
		},
		{
			name:        "boolean true",
			fileContent: `{"enabled": true}`,
			inputs:      []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "enabled", Value: "true"}},
		},
		{
			name:        "boolean false",
			fileContent: `{"enabled": false}`,
			inputs:      []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "enabled", Value: "false"}},
		},
		{
			name:        "boolean true as string",
			fileContent: `{"enabled": "true"}`,
			inputs:      []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "enabled", Value: "true"}},
		},
		{
			name:        "boolean false as string",
			fileContent: `{"enabled": "false"}`,
			inputs:      []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "enabled", Value: "false"}},
		},
		{
			name:        "select first option",
			fileContent: `{"region": "us-east-1"}`,
			inputs:      []blueprintInput{{ID: "region", Name: "Region", Type: "select", Options: []string{"us-east-1", "eu-west-1", "ap-southeast-1"}}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "region", Value: "us-east-1"}},
		},
		{
			name:        "select last option",
			fileContent: `{"region": "ap-southeast-1"}`,
			inputs:      []blueprintInput{{ID: "region", Name: "Region", Type: "select", Options: []string{"us-east-1", "eu-west-1", "ap-southeast-1"}}},
			wantPairs:   []BlueprintStackCreateInputPair{{ID: "region", Value: "ap-southeast-1"}},
		},
		{
			name: "all types in one file",
			fileContent: `{
				"txt":    "hello",
				"num":    10,
				"flt":    1.5,
				"flag":   true,
				"choice": "b"
			}`,
			inputs: []blueprintInput{
				{ID: "txt", Name: "Text", Type: "short_text"},
				{ID: "num", Name: "Num", Type: "number"},
				{ID: "flt", Name: "Float", Type: "float"},
				{ID: "flag", Name: "Flag", Type: "boolean"},
				{ID: "choice", Name: "Choice", Type: "select", Options: []string{"a", "b", "c"}},
			},
			wantPairs: []BlueprintStackCreateInputPair{
				{ID: "txt", Value: "hello"},
				{ID: "num", Value: "10"},
				{ID: "flt", Value: "1.5"},
				{ID: "flag", Value: "true"},
				{ID: "choice", Value: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeTempInputFile(t, tt.fileContent)
			got, err := inputsFromFile(filePath, tt.inputs)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPairs, got)
		})
	}
}

func TestInputsFromFile_errors(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		inputs          []blueprintInput
		wantErrContains []string
	}{
		{
			name:            "empty file",
			fileContent:     "",
			inputs:          []blueprintInput{},
			wantErrContains: []string{"failed to parse input file"},
		},
		{
			name:            "invalid JSON",
			fileContent:     `{not valid json`,
			inputs:          []blueprintInput{},
			wantErrContains: []string{"failed to parse input file"},
		},
		{
			name:            "JSON array instead of object",
			fileContent:     `["a", "b"]`,
			inputs:          []blueprintInput{},
			wantErrContains: []string{"failed to parse input file"},
		},
		{
			name:            "single missing input",
			fileContent:     `{}`,
			inputs:          []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{`missing required input "env" (Environment)`},
		},
		{
			name:        "multiple missing inputs",
			fileContent: `{}`,
			inputs: []blueprintInput{
				{ID: "env", Name: "Environment", Type: "short_text"},
				{ID: "count", Name: "Count", Type: "number"},
			},
			wantErrContains: []string{
				`missing required input "env" (Environment)`,
				`missing required input "count" (Count)`,
			},
		},
		{
			name:        "all inputs missing",
			fileContent: `{}`,
			inputs: []blueprintInput{
				{ID: "a", Name: "A", Type: "short_text"},
				{ID: "b", Name: "B", Type: "boolean"},
				{ID: "c", Name: "C", Type: "number"},
			},
			wantErrContains: []string{
				`missing required input "a"`,
				`missing required input "b"`,
				`missing required input "c"`,
			},
		},
		{
			name:            "single extra input",
			fileContent:     `{"env": "prod", "unknown": "x"}`,
			inputs:          []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{`extra input "unknown" is not defined in the blueprint`},
		},
		{
			name:        "multiple extra inputs",
			fileContent: `{"env": "prod", "foo": 1, "bar": true}`,
			inputs:      []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{
				`extra input "foo" is not defined in the blueprint`,
				`extra input "bar" is not defined in the blueprint`,
			},
		},
		{
			name:        "only extra inputs no blueprint inputs",
			fileContent: `{"ghost": "value"}`,
			inputs:      []blueprintInput{},
			wantErrContains: []string{
				`extra input "ghost" is not defined in the blueprint`,
			},
		},
		{
			name:            "short_text given number",
			fileContent:     `{"env": 42}`,
			inputs:          []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{`input "env" (Environment): must be a string`},
		},
		{
			name:            "short_text given boolean",
			fileContent:     `{"env": true}`,
			inputs:          []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{`input "env" (Environment): must be a string`},
		},
		{
			name:            "long_text given number",
			fileContent:     `{"desc": 99}`,
			inputs:          []blueprintInput{{ID: "desc", Name: "Description", Type: "long_text"}},
			wantErrContains: []string{`input "desc" (Description): must be a string`},
		},
		{
			name:            "secret given boolean",
			fileContent:     `{"tok": false}`,
			inputs:          []blueprintInput{{ID: "tok", Name: "Token", Type: "secret"}},
			wantErrContains: []string{`input "tok" (Token): must be a string`},
		},
		{
			name:            "number given float",
			fileContent:     `{"count": 3.14}`,
			inputs:          []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantErrContains: []string{`input "count" (Count): must be an integer`},
		},
		{
			name:            "number given boolean",
			fileContent:     `{"count": true}`,
			inputs:          []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantErrContains: []string{`input "count" (Count): must be an integer`},
		},
		{
			name:            "number given null",
			fileContent:     `{"count": null}`,
			inputs:          []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantErrContains: []string{`input "count" (Count): must be an integer`},
		},
		{
			name:            "float given boolean",
			fileContent:     `{"ratio": false}`,
			inputs:          []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantErrContains: []string{`input "ratio" (Ratio): must be a float`},
		},
		{
			name:            "float given null",
			fileContent:     `{"ratio": null}`,
			inputs:          []blueprintInput{{ID: "ratio", Name: "Ratio", Type: "float"}},
			wantErrContains: []string{`input "ratio" (Ratio): must be a float`},
		},
		{
			name:            "boolean given number",
			fileContent:     `{"enabled": 1}`,
			inputs:          []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantErrContains: []string{`input "enabled" (Enabled): must be a boolean`},
		},
		{
			name:            "boolean given null",
			fileContent:     `{"enabled": null}`,
			inputs:          []blueprintInput{{ID: "enabled", Name: "Enabled", Type: "boolean"}},
			wantErrContains: []string{`input "enabled" (Enabled): must be a boolean`},
		},
		{
			name:            "select invalid option",
			fileContent:     `{"region": "ap-northeast-1"}`,
			inputs:          []blueprintInput{{ID: "region", Name: "Region", Type: "select", Options: []string{"us-east-1", "eu-west-1"}}},
			wantErrContains: []string{`input "region" (Region): must be one of [us-east-1, eu-west-1], got "ap-northeast-1"`},
		},
		{
			name:            "select given number",
			fileContent:     `{"region": 1}`,
			inputs:          []blueprintInput{{ID: "region", Name: "Region", Type: "select", Options: []string{"us-east-1"}}},
			wantErrContains: []string{`input "region" (Region): must be a string`},
		},
		{
			name:            "select given boolean",
			fileContent:     `{"region": true}`,
			inputs:          []blueprintInput{{ID: "region", Name: "Region", Type: "select", Options: []string{"us-east-1"}}},
			wantErrContains: []string{`input "region" (Region): must be a string`},
		},
		{
			name:        "two type errors aggregated",
			fileContent: `{"count": "not-a-number", "flag": "not-a-bool"}`,
			inputs: []blueprintInput{
				{ID: "count", Name: "Count", Type: "number"},
				{ID: "flag", Name: "Flag", Type: "boolean"},
			},
			wantErrContains: []string{
				`input "count" (Count): must be an integer`,
				`input "flag" (Flag): must be a boolean`,
			},
		},
		{
			name:        "missing input and type error on another",
			fileContent: `{"count": "bad"}`,
			inputs: []blueprintInput{
				{ID: "env", Name: "Environment", Type: "short_text"},
				{ID: "count", Name: "Count", Type: "number"},
			},
			wantErrContains: []string{
				`missing required input "env" (Environment)`,
				`input "count" (Count): must be an integer`,
			},
		},
		{
			name:        "extra and missing inputs at the same time",
			fileContent: `{"ghost": "value"}`,
			inputs:      []blueprintInput{{ID: "env", Name: "Environment", Type: "short_text"}},
			wantErrContains: []string{
				`extra input "ghost" is not defined in the blueprint`,
				`missing required input "env" (Environment)`,
			},
		},
		{
			name:        "extra input alongside type error",
			fileContent: `{"count": "bad", "ghost": "extra"}`,
			inputs:      []blueprintInput{{ID: "count", Name: "Count", Type: "number"}},
			wantErrContains: []string{
				`extra input "ghost" is not defined in the blueprint`,
				`input "count" (Count): must be an integer`,
			},
		},
		{
			name:        "missing extra and type error all at once",
			fileContent: `{"count": true, "ghost": "extra"}`,
			inputs: []blueprintInput{
				{ID: "env", Name: "Environment", Type: "short_text"},
				{ID: "count", Name: "Count", Type: "number"},
			},
			wantErrContains: []string{
				`extra input "ghost" is not defined in the blueprint`,
				`missing required input "env" (Environment)`,
				`input "count" (Count): must be an integer`,
			},
		},
		{
			name: "multiple type errors across all types",
			fileContent: `{
				"txt":    42,
				"num":    "not-a-number",
				"flt":    "not-a-float",
				"flag":   "not-a-bool",
				"choice": "invalid-option"
			}`,
			inputs: []blueprintInput{
				{ID: "txt", Name: "Text", Type: "short_text"},
				{ID: "num", Name: "Num", Type: "number"},
				{ID: "flt", Name: "Float", Type: "float"},
				{ID: "flag", Name: "Flag", Type: "boolean"},
				{ID: "choice", Name: "Choice", Type: "select", Options: []string{"a", "b"}},
			},
			wantErrContains: []string{
				`input "txt" (Text): must be a string`,
				`input "num" (Num): must be an integer`,
				`input "flt" (Float): must be a float`,
				`input "flag" (Flag): must be a boolean`,
				`input "choice" (Choice): must be one of [a, b], got "invalid-option"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeTempInputFile(t, tt.fileContent)
			_, err := inputsFromFile(filePath, tt.inputs)
			require.Error(t, err)
			for _, substr := range tt.wantErrContains {
				assert.ErrorContains(t, err, substr)
			}
		})
	}
}
