# `spacectl api`

`spacectl api` lets you run ad-hoc read-only GraphQL queries against the Spacelift API using your existing authentication.

Mutations are not supported. For write operations, use the dedicated `spacectl` subcommands or the [Spacelift Terraform Provider](https://github.com/spacelift-io/terraform-provider-spacelift).

## Usage

Basic queries (bare field selections are wrapped in `query { ... }` automatically):

```bash
spacectl api 'stacks { id name state }'
spacectl api 'workerPools { id name workers { id } }'
```

Full query syntax:

```bash
spacectl api 'query { stack(id: "my-stack") { id name branch repository } }'
```

With variables:

```bash
spacectl api --variables '{"id":"my-stack"}' 'query($id: ID!) { stack(id: $id) { id name } }'
```

From a file or stdin:

```bash
spacectl api < query.graphql
spacectl api --variables '{"id":"my-stack"}' < query.graphql
cat query.graphql | spacectl api
```

## Output

- When stdout is a TTY, output is pretty-printed JSON.
- When piped, output is raw JSON (suitable for `jq`).
- Use `--raw` to force raw output.

```bash
spacectl api 'stacks { id name repository }' | jq '.data.stacks[] | select(.repository == "tf-infra")'
```

## Schema Introspection

You can explore the GraphQL schema using standard introspection queries:

```bash
# List all queries
spacectl api '{ __type(name: "Query") { fields { name } } }' | jq '.data.__type.fields[].name'

# Inspect a specific query's arguments and return type
spacectl api '{ __type(name: "Query") { fields { name args { name type { name kind ofType { name } } } type { name kind ofType { name } } } } }' \
  | jq '.data.__type.fields[] | select(.name == "stack")'

# Inspect a type
spacectl api '{ __type(name: "Stack") { fields { name type { name kind ofType { name } } } } }'

# List all types (excluding internals)
spacectl api '{ __schema { types { name kind } } }' | jq '[.data.__schema.types[] | select(.name | startswith("__") | not) | .name] | sort'

# List enum values
spacectl api '{ __type(name: "RunState") { enumValues { name } } }' | jq '.data.__type.enumValues[].name'

# Full schema dump (raw introspection JSON):
spacectl api '{ __schema { queryType { name } mutationType { name } types { kind name description fields(includeDeprecated: true) { name description args { name type { name kind ofType { name kind ofType { name kind } } } } type { name kind ofType { name kind ofType { name kind } } } } inputFields { name type { name kind ofType { name kind } } } enumValues { name description } possibleTypes { name } } } }' > schema.json
```
