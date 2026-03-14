package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astprinter"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
)

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
		return outputSchemaSDL(env.Data.Schema)
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

func outputSchemaSDL(schema introspection.Schema) error {
	data := introspection.Data{Schema: schema}
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
