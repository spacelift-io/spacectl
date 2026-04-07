# `spacectl stack state`

`spacectl stack state` manages Terraform/OpenTofu state files for Spacelift stacks.

## `pull`

Downloads the current state file for a stack.

```bash
# Output to stdout
spacectl stack state pull --id my-stack

# Save to file
spacectl stack state pull --id my-stack -o terraform.tfstate

# Auto-detect stack from current directory
spacectl stack state pull

# Pretty-print
spacectl stack state pull --id my-stack | jq .
```

### Prerequisites

The stack must have:

- **Manages state** enabled (Spacelift manages the Terraform state)
- **External state access** enabled (stack setting)

The user must have **State download** permission or Space admin role.
