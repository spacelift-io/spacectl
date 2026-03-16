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

Use `--schema` to dump the full GraphQL schema via introspection:

```bash
# Full schema dump
spacectl api --schema > schema.json

# List all query names
spacectl api --schema | jq '[.data.__schema.types[] | select(.name == "Query") | .fields[].name] | sort'

# List all mutation names
spacectl api --schema | jq '[.data.__schema.types[] | select(.name == "Mutation") | .fields[].name] | sort'

# Inspect a specific type
spacectl api --schema | jq '.data.__schema.types[] | select(.name == "Stack")'

# List enum values
spacectl api --schema | jq '.data.__schema.types[] | select(.name == "RunState") | .enumValues[].name'
```

You can also run targeted introspection queries directly:

```bash
# List all queries
spacectl api '{ __type(name: "Query") { fields { name } } }' | jq '.data.__type.fields[].name'

# Inspect a specific query's arguments
spacectl api '{ __type(name: "Query") { fields { name args { name type { name kind ofType { name } } } } } }' \
  | jq '.data.__type.fields[] | select(.name == "stack")'

# List all types (excluding internals)
spacectl api '{ __schema { types { name kind } } }' | jq '[.data.__schema.types[] | select(.name | startswith("__") | not) | .name] | sort'
```
