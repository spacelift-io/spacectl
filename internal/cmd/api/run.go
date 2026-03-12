package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astprinter"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type apiRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

type gqlErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type introspectionEnvelope struct {
	Data *struct {
		Schema introspection.Schema `json:"__schema"`
	} `json:"data"`
}

const introspectionQuery = `query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}`

func run(ctx context.Context, cliCmd *cli.Command) error {
	schemaOnly := cliCmd.Bool(flagSchema.Name)
	query, variables, operation, schemaSelector, err := resolveRequestParts(cliCmd, schemaOnly)
	if err != nil {
		return err
	}

	reqBody := apiRequest{
		Query:         query,
		Variables:     variables,
		OperationName: operation,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/graphql", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient, token, endpoint, err := authenticated.HTTPClient(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	baseURL, err := url.Parse(strings.TrimRight(endpoint, "/graphql"))
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}
	req.URL.Scheme = baseURL.Scheme
	req.URL.Host = baseURL.Host

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if schemaOnly {
		if err := outputSchema(body, schemaSelector); err != nil {
			return err
		}
	} else {
		if err := outputResponse(body, cliCmd.Bool(flagRaw.Name)); err != nil {
			return err
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthorized: you can re-login using `spacectl profile login`")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, snippet(body))
	}

	if msg, ok := graphqlErrorMessage(body); ok {
		return fmt.Errorf("graphql errors: %s", msg)
	}

	return nil
}

func resolveRequestParts(cliCmd *cli.Command, schemaOnly bool) (string, map[string]any, string, string, error) {
	query := strings.TrimSpace(cliCmd.String(flagQuery.Name))
	file := strings.TrimSpace(cliCmd.String(flagFile.Name))
	variablesRaw := cliCmd.String(flagVariables.Name)
	operation := strings.TrimSpace(cliCmd.String(flagOperation.Name))
	argsQuery := strings.TrimSpace(strings.Join(cliCmd.Args().Slice(), " "))

	if err := validateSchemaArgs(schemaOnly, query, file, variablesRaw, operation, argsQuery); err != nil {
		return "", nil, "", "", err
	}

	if schemaOnly {
		return introspectionQuery, nil, "", argsQuery, nil
	}

	if query == "" && file == "" && argsQuery != "" {
		return normalizeQuery(argsQuery), nil, operation, "", nil
	}

	resolvedQuery, err := resolveQueryFrom(query, file, os.Stdin, isatty.IsTerminal(os.Stdin.Fd()))
	if err != nil {
		return "", nil, "", "", err
	}

	variables, err := parseVariables(variablesRaw)
	if err != nil {
		return "", nil, "", "", err
	}

	return resolvedQuery, variables, operation, "", nil
}

func resolveQueryFrom(query string, file string, stdin io.Reader, stdinIsTerminal bool) (string, error) {
	if query != "" && file != "" {
		return "", errors.New("only one of --query or --file can be specified")
	}

	if query == "-" {
		return readQueryFromStdin(stdin)
	}

	if query != "" {
		return query, nil
	}

	if file != "" {
		contents, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return strings.TrimSpace(string(contents)), nil
	}

	if !stdinIsTerminal {
		return readQueryFromStdin(stdin)
	}

	return "", errors.New("query required: use --query, --file, or stdin")
}

func readQueryFromStdin(stdin io.Reader) (string, error) {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	query := strings.TrimSpace(string(data))
	if query == "" {
		return "", errors.New("stdin is empty")
	}
	return query, nil
}

func parseVariables(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("failed to parse variables JSON: %w", err)
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("variables must be a JSON object")
	}

	return obj, nil
}

func validateSchemaArgs(schemaOnly bool, query string, file string, variables string, operation string, argsQuery string) error {
	if !schemaOnly {
		return nil
	}

	if query != "" || file != "" || strings.TrimSpace(variables) != "" || operation != "" {
		return errors.New("--schema cannot be combined with query, file, variables, or operation flags")
	}

	return nil
}

func outputResponse(body []byte, raw bool) error {
	if raw || !isatty.IsTerminal(os.Stdout.Fd()) {
		_, err := os.Stdout.Write(body)
		return err
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		_, werr := os.Stdout.Write(body)
		return werr
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func outputSchemaSDL(body []byte) error {
	var env introspectionEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("failed to parse schema response: %w", err)
	}
	if env.Data == nil {
		return errors.New("missing schema data")
	}

	data := introspection.Data{Schema: env.Data.Schema}
	encoded, err := json.Marshal(&data)
	if err != nil {
		return fmt.Errorf("failed to encode introspection schema: %w", err)
	}

	converter := introspection.JsonConverter{}
	document, err := converter.GraphQLDocument(bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("failed to parse introspection schema: %w", err)
	}

	var buff bytes.Buffer
	if err := astprinter.PrintIndent(document, []byte("  "), &buff); err != nil {
		return err
	}

	doc, err := parser.ParseSchema(&ast.Source{Name: "schema", Input: buff.String()})
	if err != nil {
		return fmt.Errorf("failed to format schema: %w", err)
	}

	var formatted bytes.Buffer
	formatter.NewFormatter(&formatted, formatter.WithIndent("  ")).FormatSchemaDocument(doc)
	output := addTopLevelSpacing(formatted.String())
	_, err = os.Stdout.Write([]byte(output))
	return err
}

func outputSchema(body []byte, selector string) error {
	var env introspectionEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("failed to parse schema response: %w", err)
	}
	if env.Data == nil {
		return errors.New("missing schema data")
	}

	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return outputSchemaSDL(body)
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case "queries", "query":
		return listSchemaFields(env.Data.Schema, env.Data.Schema.QueryType.Name)
	case "mutations", "mutation":
		if env.Data.Schema.MutationType == nil {
			return errors.New("schema has no mutations")
		}
		return listSchemaFields(env.Data.Schema, env.Data.Schema.MutationType.Name)
	case "types", "type":
		return listSchemaTypes(env.Data.Schema)
	default:
		return outputSchemaInfo(env.Data.Schema, trimmed)
	}
}

func graphqlErrorMessage(body []byte) (string, bool) {
	var gqlErr gqlErrorResponse
	if err := json.Unmarshal(body, &gqlErr); err != nil {
		return "", false
	}
	if len(gqlErr.Errors) == 0 {
		return "", false
	}
	if gqlErr.Errors[0].Message != "" {
		return gqlErr.Errors[0].Message, true
	}
	return "unknown error", true
}

func snippet(body []byte) string {
	const limit = 512
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) <= limit {
		return string(trimmed)
	}
	return string(trimmed[:limit]) + "..."
}

func normalizeQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "query") || strings.HasPrefix(lower, "mutation") || strings.HasPrefix(lower, "subscription") || strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	return "query { " + trimmed + " }"
}

func listSchemaFields(schema introspection.Schema, typeName string) error {
	typeDef := findType(schema, typeName)
	if typeDef == nil {
		return fmt.Errorf("type %q not found in schema", typeName)
	}

	names := make([]string, 0, len(typeDef.Fields))
	for _, field := range typeDef.Fields {
		names = append(names, field.Name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(os.Stdout, name)
	}

	return nil
}

func listSchemaTypes(schema introspection.Schema) error {
	names := make([]string, 0, len(schema.Types))
	for _, t := range schema.Types {
		if t != nil && t.Name != "" {
			names = append(names, t.Name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(os.Stdout, name)
	}

	return nil
}

func outputSchemaInfo(schema introspection.Schema, name string) error {
	queryType := findType(schema, schema.QueryType.Name)
	if field := findField(queryType, name); field != nil {
		return printFieldInfo("query", field)
	}

	if schema.MutationType != nil {
		mutationType := findType(schema, schema.MutationType.Name)
		if field := findField(mutationType, name); field != nil {
			return printFieldInfo("mutation", field)
		}
	}

	typeDef := findType(schema, name)
	if typeDef == nil {
		return fmt.Errorf("no query, mutation, or type named %q found", name)
	}

	return printTypeInfo(typeDef)
}

func findType(schema introspection.Schema, name string) *introspection.FullType {
	for _, t := range schema.Types {
		if t != nil && t.Name == name {
			return t
		}
	}
	return nil
}

func findField(typeDef *introspection.FullType, name string) *introspection.Field {
	if typeDef == nil {
		return nil
	}
	for i := range typeDef.Fields {
		if typeDef.Fields[i].Name == name {
			return &typeDef.Fields[i]
		}
	}
	return nil
}

func printFieldInfo(kind string, field *introspection.Field) error {
	if field == nil {
		return errors.New("field not found")
	}

	signature := field.Name
	if len(field.Args) > 0 {
		args := make([]string, 0, len(field.Args))
		for _, arg := range field.Args {
			argSig := fmt.Sprintf("%s: %s", arg.Name, formatTypeRef(arg.Type))
			if arg.DefaultValue != nil && strings.TrimSpace(*arg.DefaultValue) != "" {
				argSig = argSig + " = " + strings.TrimSpace(*arg.DefaultValue)
			}
			args = append(args, argSig)
		}
		signature = fmt.Sprintf("%s(%s)", signature, strings.Join(args, ", "))
	}
	signature = fmt.Sprintf("%s: %s", signature, formatTypeRef(field.Type))

	fmt.Fprintf(os.Stdout, "%s %s\n", kind, signature)
	if len(field.Args) > 0 {
		fmt.Fprintln(os.Stdout, "Args:")
		for _, arg := range field.Args {
			line := fmt.Sprintf("%s: %s", arg.Name, formatTypeRef(arg.Type))
			if arg.DefaultValue != nil && strings.TrimSpace(*arg.DefaultValue) != "" {
				line = line + " = " + strings.TrimSpace(*arg.DefaultValue)
			}
			fmt.Fprintln(os.Stdout, line)
		}
	}

	return nil
}

func printTypeInfo(typeDef *introspection.FullType) error {
	if typeDef == nil {
		return errors.New("type not found")
	}

	fmt.Fprintf(os.Stdout, "%s (%s)\n", typeDef.Name, typeDef.Kind.String())

	switch typeDef.Kind {
	case introspection.OBJECT, introspection.INTERFACE:
		fmt.Fprintln(os.Stdout, "Fields:")
		for _, field := range typeDef.Fields {
			fmt.Fprintf(os.Stdout, "%s: %s\n", field.Name, formatTypeRef(field.Type))
		}
	case introspection.INPUTOBJECT:
		fmt.Fprintln(os.Stdout, "Input fields:")
		for _, field := range typeDef.InputFields {
			fmt.Fprintf(os.Stdout, "%s: %s\n", field.Name, formatTypeRef(field.Type))
		}
	case introspection.ENUM:
		fmt.Fprintln(os.Stdout, "Values:")
		for _, value := range typeDef.EnumValues {
			fmt.Fprintln(os.Stdout, value.Name)
		}
	case introspection.UNION:
		fmt.Fprintln(os.Stdout, "Possible types:")
		for _, t := range typeDef.PossibleTypes {
			fmt.Fprintln(os.Stdout, formatTypeRef(t))
		}
	case introspection.SCALAR:
		if typeDef.SpecifiedByURL != nil && *typeDef.SpecifiedByURL != "" {
			fmt.Fprintf(os.Stdout, "Specified by: %s\n", *typeDef.SpecifiedByURL)
		}
	}

	return nil
}

func formatTypeRef(t introspection.TypeRef) string {
	switch t.Kind {
	case introspection.NONNULL:
		if t.OfType == nil {
			return "Unknown!"
		}
		return formatTypeRef(*t.OfType) + "!"
	case introspection.LIST:
		if t.OfType == nil {
			return "[Unknown]"
		}
		return "[" + formatTypeRef(*t.OfType) + "]"
	default:
		if t.Name != nil {
			return *t.Name
		}
		if t.OfType != nil {
			return formatTypeRef(*t.OfType)
		}
		return "Unknown"
	}
}

func addTopLevelSpacing(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	previousNonEmpty := false
	previousWasTopLevelComment := false
	inTopLevelBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		isTopLevelComment := isTopLevelCommentLine(line, trimmed, &inTopLevelBlockComment)
		isTopLevelDefinition := isTopLevelDefinitionLine(line, trimmed)
		isTopLevelLine := isTopLevelComment || isTopLevelDefinition

		if isTopLevelLine && previousNonEmpty && !previousWasTopLevelComment {
			out = append(out, "")
		}

		out = append(out, line)
		previousNonEmpty = true
		previousWasTopLevelComment = isTopLevelComment || inTopLevelBlockComment
	}

	return strings.Join(out, "\n") + "\n"
}

func isTopLevelCommentLine(line string, trimmed string, inBlock *bool) bool {
	if strings.TrimLeft(line, " \t") != line {
		return false
	}

	if *inBlock {
		if trimmed == "\"\"\"" {
			*inBlock = false
		}
		return true
	}

	if trimmed == "\"\"\"" {
		*inBlock = true
		return true
	}

	if strings.HasPrefix(trimmed, "\"\"\"") && strings.HasSuffix(trimmed, "\"\"\"") && len(trimmed) > len("\"\"\"\"\"\"") {
		return true
	}

	if strings.HasPrefix(trimmed, "\"\"\"") {
		*inBlock = true
		return true
	}

	if strings.HasPrefix(trimmed, "\"") {
		return true
	}

	return false
}

func isTopLevelDefinitionLine(line string, trimmed string) bool {
	if strings.TrimLeft(line, " \t") != line {
		return false
	}

	switch {
	case strings.HasPrefix(trimmed, "schema "):
		return true
	case strings.HasPrefix(trimmed, "type "):
		return true
	case strings.HasPrefix(trimmed, "interface "):
		return true
	case strings.HasPrefix(trimmed, "input "):
		return true
	case strings.HasPrefix(trimmed, "union "):
		return true
	case strings.HasPrefix(trimmed, "enum "):
		return true
	case strings.HasPrefix(trimmed, "scalar "):
		return true
	case strings.HasPrefix(trimmed, "directive "):
		return true
	case strings.HasPrefix(trimmed, "extend "):
		return true
	default:
		return false
	}
}
