# `spacectl`, the Spacelift CLI

`spacectl` is a utility wrapping Spacelift's [GraphQL API](https://docs.spacelift.io/integrations/api) for easy programmatic access in command-line contexts - either in manual interactive mode (in your local shell), or in a predefined CI pipeline (GitHub actions, CircleCI, Jenkins etc).

Its primary purpose is to help you explore and execute actions inside Spacelift. It provides limited functionality for creating or editing resources. To do that programatically, you can use the [Spacelift Terraform Provider](https://github.com/spacelift-io/terraform-provider-spacelift).

## Installation

### Officially supported packages

Officially supported packages are maintained by [Spacelift](https://spacelift.io/) and are the preferred ways to install `spacectl`

#### Homebrew

You can install `spacectl` using Homebrew on MacOS or Linux:

```bash
brew install spacelift-io/spacelift/spacectl
```

#### Windows

You can install `spacectl` using winget:

```shell
winget install spacectl
```

or

```shell
winget install --id spacelift-io.spacectl
```

#### Docker image

`spacectl` is distributed as a Docker image, which can be used as follows:

```bash
docker run -it --rm ghcr.io/spacelift-io/spacectl stack deploy --id my-infra-stack
```

> Don't forget to add the [required environment variables](#authenticating-using-environment-variables) in order to authenticate.

#### asdf

```bash
asdf plugin add spacectl
asdf install spacectl latest
asdf global spacectl latest
```

#### GitHub Release

Alternatively, `spacectl` is distributed through GitHub Releases as a zip file containing a self-contained statically linked executable built from the source in this repository. Binaries can be download directly from the [Releases page](https://github.com/spacelift-io/spacectl/releases).

#### Usage on GitHub Actions

We have [setup-spacectl](https://github.com/spacelift-io/setup-spacectl) GitHub Action that can be used to install `spacectl`:

```yaml
steps:
  - name: Install spacectl
    uses: spacelift-io/setup-spacectl@main

  - name: Deploy infrastructure
    env:
      SPACELIFT_API_KEY_ENDPOINT: https://mycorp.app.spacelift.io
      SPACELIFT_API_KEY_ID: ${{ secrets.SPACELIFT_API_KEY_ID }}
      SPACELIFT_API_KEY_SECRET: ${{ secrets.SPACELIFT_API_KEY_SECRET }}
    run: spacectl stack deploy --id my-infra-stack
```

---

### Community supported packages

**Disclaimer:** These packages are community-maintained, please verify the package integrity yourself before using them to install or update `spacectl`.

#### Arch linux

Install [`spacectl-bin`](https://aur.archlinux.org/packages/spacectl-bin) from the Arch User Repository ([AUR](https://aur.archlinux.org/)):

```bash
yay -S spacectl-bin
```

Please make sure to verify the [`PKGBUILD`](https://aur.archlinux.org/cgit/aur.git/tree/PKGBUILD?h=spacectl-bin) before installing/updating.

#### Alpine linux

Install [`spacectl`](https://pkgs.alpinelinux.org/packages?name=spacectl&branch=edge&repo=&arch=&maintainer=) from the Alpine Repository ([alpine packages](https://pkgs.alpinelinux.org/packages)):

```bash
apk add spacectl --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing
```

Please make sure to verify the [`APKBUILD`](https://git.alpinelinux.org/aports/tree/testing/spacectl/APKBUILD) before installing/updating.

## Quick Start

Authenticate using `spacectl profile login`:

```bash
> spacectl profile login my-account
Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/): http://my-account.app.spacelift.tf
Select authentication flow:
  1) for API key,
  2) for GitHub access token,
  3) for login with a web browser
Option: 3
```

Use spacectl :rocket::

```bash
> spacectl stack list
Name                          | Commit   | Author        | State     | Worker Pool | Locked By
stack-1                       | 1aa0ef62 | Adam Connelly | NONE      |             |
stack-2                       | 1aa0ef62 | Adam Connelly | DISCARDED |             |
```

## Getting Help

To list all the commands available, use `spacectl help`:

```bash
> spacectl help
NAME:
   spacectl - Programmatic access to Spacelift GraphQL API.

USAGE:
   spacectl [global options] [command [command options]]

VERSION:
   1.13.0

COMMANDS:
   profile                  Manage Spacelift profiles
   whoami                   Print out logged-in user's information
   version                  Print out CLI version
   module                   Manage a Spacelift module
   stack                    Manage a Spacelift stack
   provider                 Manage a Terraform provider
   run-external-dependency  Manage Spacelift Run external dependencies
   workerpool               Manages workerpools and their workers.
   blueprint                Manage Spacelift blueprints
   policy                   Manage Spacelift policies
   audit-trail              Manage Spacelift audit trail entries
   mcp                      Manage MCP server
   help, h                  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

To get help about a particular command or subcommand, use the `-h` flag:

```bash
> spacectl profile -h
NAME:
   spacectl profile - Manage Spacelift profiles

USAGE:
   spacectl profile [command [command options]] 

COMMANDS:
   current       Outputs your currently selected profile
   export-token  Prints the current token to stdout. In order not to leak, we suggest piping it to your OS pastebin
   usage-csv     Prints CSV with usage data for the current account
   list          List all your Spacelift account profiles
   login         Create a profile for a Spacelift account
   logout        Remove Spacelift credentials for an existing profile
   select        Select one of your Spacelift account profiles

OPTIONS:
   --help, -h  show help
```

## Example

The following screencast shows an example of using spacectl to run a one-off task in Spacelift:

[![asciicast](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B.svg)](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B)

## Authentication

`spacectl` is designed to work in two different contexts - a non-interactive scripting mode (eg. external CI/CD pipeline) and a local interactive mode, where you type commands into your shell. Because of this, it supports two types of credentials - environment variables and user profiles.

We refer to each method of providing credentials as "credential providers" (like AWS), and details of each method are documented in the following sections.

### Authenticating using environment variables

The CLI supports the following authentication methods via the environment:

- [Spacelift API tokens](#spacelift-api-tokens).
- [GitHub tokens](#github-tokens).
- [Spacelift API keys](#spacelift-api-keys).

`spacectl` looks for authentication configurations in the order specified above, and will stop as soon as it finds a valid configuration. For example, if a Spacelift API token is specified, GitHub tokens and Spacelift API keys will be ignored, even if their environment variables are specified.

#### Spacelift API tokens

Spacelift API tokens can be specified using the `SPACELIFT_API_TOKEN` environment variable. When this variable is found, the CLI ignores all the other authentication environment variables because the token contains all the information needed to authenticate.

NOTE: API tokens are generally short-lived and will need to be re-created often.

#### GitHub tokens

GitHub tokens are only available to accounts that use GitHub as their identity provider, but are very convenient for use in GitHub actions. To use a GitHub token, set the following environment variables:

- `SPACELIFT_API_KEY_ENDPOINT` - the URL to your Spacelift account, for example `https://mycorp.app.spacelift.io`.
- `SPACELIFT_API_GITHUB_TOKEN` - a GitHub personal access token.

#### Spacelift API keys

To use a Spacelift API key, set the following environment variables:

- `SPACELIFT_API_KEY_ENDPOINT` - the URL to your Spacelift account, for example `https://mycorp.app.spacelift.io`.
- `SPACELIFT_API_KEY_ID` - the ID of your Spacelift API key. Available via the Spacelift application.
- `SPACELIFT_API_KEY_SECRET` - the secret for your API key. Only available when the secret is created.

More information about API authentication can be found at <https://docs.spacelift.io/integrations/api#authenticating-with-the-api>.

### Authenticating using account profiles

In order to make working with multiple Spacelift accounts easy in interactive scenarios, Spacelift supports account management through the `profile` family of commands:

```bash
❯ spacectl profile
NAME:
   spacectl profile - Manage Spacelift profiles

USAGE:
   spacectl profile command [command options] [arguments...]

COMMANDS:
     current       Outputs your currently selected profile
     export-token  Prints the current token to stdout. In order not to leak, we suggest piping it to your OS pastebin
     list          List all your Spacelift account profiles
     login         Create a profile for a Spacelift account
     logout        Remove Spacelift credentials for an existing profile
     select        Select one of your Spacelift account profiles
     help, h       Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

Each of the subcommands requires an account **alias**, which is a short, user-friendly name for each set of credentials (account profiles). Profiles don't need to be unique - you can have multiple sets of credentials for a single account too.

Account profiles support three authentication methods:

- GitHub access tokens
- API keys
- Login with a browser (API token).

In order to authenticate to your first profile, type in the following (make sure to replace `${MY_ALIAS}` with the actual profile alias):

```bash
❯ spacectl profile login ${MY_ALIAS}
Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/):
```

In the next step, you will be asked to choose which authentication method you are going to use. Note that if your account is using [SAML-based SSO authentication](https://docs.spacelift.io/integrations/single-sign-on), then API keys and login with a browser are your only options. After you're done entering credentials, the CLI will validate them against the server, and assuming that they're valid, will persist them in a credentials file in `.spacelift/${MY_ALIAS}`. It will also create a symlink in `${HOME}/.spacelift/current` pointing to the current profile.

By default the login process is interactive, however, if that does not fit your workflow, the steps can be predefined using flags, for example:

```bash
❯ spacectl profile login --method browser --endpoint https://unicorn.app.spacelift.io local-test
```

You can switch between account profiles by using `spacectl profile select ${MY_ALIAS}`. What this does behind the scenes is point `${HOME}/.spacelift/current` to the new location. You can also delete stored credetials for a given profile by using the `spacectl profile logout ${MY_ALIAS}` command.

## MCP Server

Spacectl includes an MCP (Model Context Protocol) server that allows AI models to interact with Spacelift through a standardized interface. MCP is an open protocol that standardizes how applications provide context to LLMs, similar to how USB-C provides a standardized way to connect devices to peripherals.

### Authentication

The MCP server uses the same authentication methods as the standard CLI. You can use any of the [authentication methods described above](#authentication).

### Configuration

To use the Spacelift MCP server with your AI tool:

1. Find your tool's MCP configuration file
2. Add the following configuration:

```json
{
  "mcpServers": {
    "spacelift": {
      "command": "spacectl",
      "args": ["mcp", "server"]
    }
  }
}
```

Or if you prefer using Docker:

```json
{
  "mcpServers": {
    "spacelift": {
      "command": "docker",
      "args": [
        "run", 
        "-i", 
        "--rm", 
        "-e", "SPACELIFT_API_TOKEN=your-api-token-here",
        // Or use API key authentication:
        // "-e", "SPACELIFT_API_KEY_ENDPOINT=https://your-account.app.spacelift.io",
        // "-e", "SPACELIFT_API_KEY_ID=your-key-id",
        // "-e", "SPACELIFT_API_KEY_SECRET=your-key-secret",
        "ghcr.io/spacelift-io/spacectl", 
        "mcp", 
        "server"
      ]
    }
  }
}
```

3. Restart your AI tool to apply the changes

### Available Tools

The MCP server provides several tools for AI assistants to interact with Spacelift:

- **list_stacks**: Browse and search through your Spacelift stacks
- **list_stack_runs**: View the run history for a specific stack
- **get_stack_run_logs**: Access detailed logs for a specific run
- **get_stack_run_changes**: See infrastructure changes from a run
- **trigger_stack_run**: Start a new run for a stack
- **confirm_stack_run**: Approve a run that is waiting for confirmation
- **discard_stack_run**: Cancel a pending or in-progress run
- **list_resources**: View infrastructure resources managed by your stacks
- **local_preview**: Create a preview run using local workspace files

With these tools, AI assistants can securely access your Spacelift infrastructure data and perform operations on your behalf.

## Contributing

For information about how to contribute to the development of spacectl, please see our [CONTRIBUTING.md](./CONTRIBUTING.md) file.
