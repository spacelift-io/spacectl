---
name: spacectl
description: Manage Spacelift stacks, runs, modules, policies, and infrastructure via CLI.
allowed-tools: Bash(spacectl:*)
---

# Spacelift CLI (spacectl)

## Quick start

```bash
# check current profile and auth
spacectl profile current
spacectl whoami
# list stacks
spacectl stack list
# deploy a stack
spacectl stack deploy --id my-stack --tail
# preview changes from local workspace
spacectl stack local-preview --id my-stack
```

## Commands

### Authentication

Always operate on the current profile. Use `select` to switch, `login` (no alias) to re-authenticate. Login opens a browser for SSO.

```bash
spacectl profile current
spacectl whoami
spacectl profile list
spacectl profile select <account-alias>
# re-authenticate current profile (opens browser)
spacectl profile login
spacectl profile export-token
```

### Stack — Inspection

```bash
spacectl stack list
spacectl stack list -o json
spacectl stack list --search "production" --limit 10
spacectl stack list --show-labels
spacectl stack show --id my-stack
spacectl stack show --id my-stack -o json
spacectl stack outputs --id my-stack
spacectl stack outputs --id my-stack --output-id specific_output
spacectl stack resources list --id my-stack
spacectl stack run list --id my-stack
spacectl stack run list --id my-stack --preview-runs --max-results 20
spacectl stack dependencies on --id my-stack
spacectl stack dependencies off --id my-stack
```

### Stack — Run Management

Run states: QUEUED -> PREPARING -> PLANNING -> UNCONFIRMED -> APPLYING -> FINISHED (or FAILED/DISCARDED/CANCELED).

```bash
# deploy (tracked run)
spacectl stack deploy --id my-stack --tail
spacectl stack deploy --id my-stack --sha abc123 --tail
spacectl stack deploy --id my-stack --auto-confirm
spacectl stack deploy --id my-stack --runtime-config runtime.yaml --tail
# preview (proposed run, plan only)
spacectl stack preview --id my-stack --tail
spacectl stack preview --id my-stack --sha abc123
# run lifecycle
spacectl stack confirm --id my-stack --run 01JRUN123 --tail
spacectl stack discard --id my-stack --run 01JRUN123
spacectl stack cancel --id my-stack --run 01JRUN123
spacectl stack retry --id my-stack --run 01JRUN123 --tail
spacectl stack replan --id my-stack --run 01JRUN123 --tail
spacectl stack replan --id my-stack --run 01JRUN123 --resources "aws_instance.foo"
spacectl stack prioritize --id my-stack --run 01JRUN123
spacectl stack deprioritize --id my-stack --run 01JRUN123
# approval
spacectl stack approve --id my-stack --run 01JRUN123 --note "LGTM"
spacectl stack reject --id my-stack --run 01JRUN123 --note "needs fix"
# approve current stack blocker (no specific run)
spacectl stack approve --id my-stack
# logs and changes
spacectl stack logs --id my-stack --run 01JRUN123
spacectl stack logs --id my-stack --run-latest
spacectl stack changes --id my-stack --run 01JRUN123
# task (arbitrary command in stack environment)
spacectl stack task --id my-stack "terraform state list" --tail
spacectl stack task --id my-stack --noinit "echo hello" --tail
```

### Stack — Local Preview

Packages local files (respects `.gitignore` and `.terraformignore`), uploads to Spacelift, triggers a preview run.

```bash
# auto-detects stack from git repo
spacectl stack local-preview
# explicit stack
spacectl stack local-preview --id my-stack
# env var overrides
spacectl stack local-preview --id my-stack --env-var-override "FOO=bar"
spacectl stack local-preview --id my-stack --tf-env-var-override "var_name=value"
# target specific resources
spacectl stack local-preview --id my-stack --target "aws_instance.foo" --target "aws_s3_bucket.bar"
# package only project root (not entire repo)
spacectl stack local-preview --id my-stack --project-root-only
# debug — create archive without uploading
spacectl stack local-preview --id my-stack --no-upload
# prioritize the preview run
spacectl stack local-preview --id my-stack --prioritize-run
# suppress log tailing
spacectl stack local-preview --id my-stack --no-tail
```

### Stack — Management

```bash
spacectl stack open --id my-stack
spacectl stack open --id my-stack --run 01JRUN123
spacectl stack lock --id my-stack --note "deploying manually"
spacectl stack unlock --id my-stack
spacectl stack enable --id my-stack
spacectl stack disable --id my-stack
spacectl stack set-current-commit --id my-stack --sha abc123
spacectl stack sync-commit --id my-stack
spacectl stack delete --id my-stack --skip-confirmation
spacectl stack delete --id my-stack --destroy-resources --skip-confirmation
```

### Stack — Environment Variables

```bash
spacectl stack environment list --id my-stack
spacectl stack environment list --id my-stack -o json
spacectl stack environment setvar --id my-stack MY_VAR "my-value"
# write-only: not readable outside runs
spacectl stack environment setvar --id my-stack SECRET_VAR "s3cret" --write-only
spacectl stack environment mount --id my-stack config.tfvars ./local-file.tfvars
spacectl stack environment mount --id my-stack config.tfvars --write-only < file.tfvars
spacectl stack environment delete --id my-stack MY_VAR
```

### Modules

```bash
spacectl module list
spacectl module list -o json --search "vpc"
spacectl module list-versions --id my-module
spacectl module create-version --id my-module --version "1.2.3" --sha abc123
spacectl module delete-version --id my-module --version-id 01JVER123
spacectl module local-preview --id my-module
spacectl module local-preview --id my-module --tests
```

### Policies

```bash
spacectl policy list
spacectl policy list --search "plan" --limit 5
spacectl policy show --id my-policy
spacectl policy samples --id my-policy
spacectl policy sample --id my-policy --key "sample-key"
spacectl policy simulate --id my-policy --input '{"key": "value"}'
spacectl policy simulate --id my-policy --input input.json
```

### Blueprints

```bash
spacectl blueprint list
spacectl blueprint show --id my-blueprint
spacectl blueprint deploy --b-id my-blueprint
```

### Worker Pools

```bash
spacectl workerpool list
spacectl workerpool list -o json
spacectl workerpool worker list --pool-id 01JPOOL123
spacectl workerpool worker drain --id 01JWORKER123 --pool-id 01JPOOL123
spacectl workerpool worker drain --id 01JWORKER123 --pool-id 01JPOOL123 --wait-until-drained
spacectl workerpool worker undrain --id 01JWORKER123 --pool-id 01JPOOL123
spacectl workerpool worker cycle --pool-id 01JPOOL123
```

### Providers

```bash
spacectl provider add-gpg-key --name "release-key" --generate
spacectl provider add-gpg-key --name "release-key" --import --path key.gpg
spacectl provider list-gpg-keys
spacectl provider revoke-gpg-key --id 01JKEY123
spacectl provider create-version --type my-provider --gpg-key-id 01JKEY123
spacectl provider list-versions --type my-provider
spacectl provider publish-version --version-id 01JVER123
spacectl provider revoke-version --version-id 01JVER123
spacectl provider delete-version --version-id 01JVER123
```

### Other

```bash
# audit trail
spacectl audit-trail list
spacectl audit-trail list --search "stack" --limit 20 -o json
# external dependencies
spacectl run-external-dependency mark-completed --id dep-123 --status finished
spacectl run-external-dependency mark-completed --id dep-123 --status failed
# usage data
spacectl profile usage-csv --since 2025-01-01 --until 2025-03-31
spacectl profile usage-csv --aspect run-minutes --group-by run-state --file usage.csv
# version
spacectl version
```

## Output formats

All list/show commands support `--output` (`-o`) with values `table` (default) or `json`. Use `--no-color` to disable ANSI colors (auto-disabled when piped).

```bash
spacectl stack list -o json
spacectl stack show --id my-stack -o json
spacectl stack outputs --id my-stack -o json
spacectl workerpool list -o json | jq '.[].name'
```

## Smart stack selection

When `--id` is omitted, spacectl auto-detects the stack from the current git repository and subdirectory. If multiple stacks match, an interactive prompt appears. Set `SPACECTL_SKIP_STACK_PROMPT=true` to auto-select.

```bash
# from inside a git repo linked to a Spacelift stack
spacectl stack show
spacectl stack deploy --tail
spacectl stack local-preview
```

## Example: full deploy flow

```bash
spacectl stack deploy --id my-stack --tail
# wait for UNCONFIRMED, review the plan output, then:
spacectl stack confirm --id my-stack --run 01JRUN123 --tail
# or discard if something looks wrong:
spacectl stack discard --id my-stack --run 01JRUN123
```

## Example: auto-confirm deploy

```bash
spacectl stack deploy --id my-stack --auto-confirm
```
