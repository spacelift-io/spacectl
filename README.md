# Spacelift CLI

Spacelift CLI (`spacelift-cli`) is a utility wrapping Spacelift's [GraphQL API](https://docs.spacelift.io/integrations/api) for easy programmatic access in command-line contexts - either in manual interactive mode (in your local shell), or in a predefined CI pipeline (GitHub actions, CircleCI, Jenkins etc).

## Installation

Spacelift CLI is distributed through GitHub Releases as a gzipped tarball containing a self-contained statically linked executable built from the source in this repository. Binaries can be download directly from the [Releases page](https://github.com/spacelift-io/spacelift-cli/releases) or programmatically using `curl` or `wget`. The example below covers macOS (Intel CPU) installation of the latest available release. In order to download and install a Linux version, change `darwin` to `linux`. If you're using an ARM processor (eg. Raspberry Pi, Apple Silicon), change `amd64` to `arm64`:

```bash
curl -s -L https://github.com/spacelift-io/spacelift-cli/releases/latest/download/spacelift-cli-darwin-amd64.tar.gz | tar xz -C /tmp && \
mv /tmp/spacelift-cli-darwin-amd64 /usr/local/bin/spacelift-cli
```

## Authentication

Spacelift CLI is designed to work in two different contexts - a non-interactive scripting mode (eg. external CI/CD pipeline) and a local interactive mode, where you type commands into your shell. Because of this, it supports two types of credentials - environment variables and user profiles.

We refer to each method of providing credentials as "credential providers" (like AWS), and details of each method are documented in the following sections.

### Authenticating using environment variables

When certain variables are defined in the environment, `spacelift-cli` will try to use them as Spacelift API credentials. First, the `SPACELIFT_API_TOKEN`. IF this variable is present in the environment, it alone contains enough information to set up a Spacelift session. Hence, the CLI will look no further and attempt to use it to talk to the server. Note that these tokens are generally short-lived.

If the `SPACELIFT_API_TOKEN` variable is not present, the CLI will look for the
`SPACELIFT_API_ENDPOINT` variable. This one contains the API endpoint to talk to, essentially pointing to your account and (optionally) Spacelift server - for example, `https://mycorp.app.spacelift.io`. If this variable is not present, environment lookup fails. Otherwise, it proceeds to credentials lookup.

First, it looks for `SPACELIFT_API_GITHUB_TOKEN`. While only avaialble to accounts that use GitHub as their identity provider, this environment-based authentication method is very convenient in the context of GitHub Actions. You can read more about this approach [here](https://docs.spacelift.io/integrations/api#authenticating-with-the-api). If this variable is available, the credentials lookup terminates. Otherwise, it assumes API keys are used.

API key credentials (id and secret) need to be provided through the `SPACELIFT_API_KEY_ID` and `SPACELIFT_API_KEY_SECRET` environment variables, respectively. You can read more about generating them [here](https://docs.spacelift.io/integrations/api#api-key-management).

### Authenticating using account profiles

In order to make working with multiple Spacelift accounts easy in interactive scenarios, Spacelift supports account management through the `account` family of commands:

```bash
❯ spacelift-cli account --help
NAME:
   spacelift-cli account - Manage Spacelift accounts

USAGE:
   spacelift-cli account command [command options] [arguments...]

COMMANDS:
   login    Log in to a Spacelift account
   logout   Log out of an existing Spacelift account
   select   Select one of your Spacelift accounts
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

Each of the subcommands requires an account **alias**, which is a short, user-friendly name for each set of credentials (account profiles). Profiles don't need to be unique - you can have multiple sets of credentials for a single account too.

Account profiles don't use short-lived tokens, so GitHub access tokens and API keys are the only two supported authentication methods. In order to authenticate to your first account, type in the following (make sure to replace `${MY_ALIAS}` with the actual account alias):

```bash
❯ spacelift-cli account login ${MY_ALIAS}
Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/):
```

In the next step, you will be asked to choose which authentication method you are going to use. Note that if your account is using [SAML-based SSO authentication](https://docs.spacelift.io/integrations/single-sign-on), then API keys are your only option. After you're done entering credentials, the CLI will validate them against the server, and assuming that they're valid, will persist them in a credentials file in `.spacelift/${MY_ALIAS}`. It will also create a symlink in `${HOME}/.spacelift/current` pointing to the current profile.

You can switch between account profiles by using `spacelift-cli account select ${MY_ALIAS}`. What this does behind the scenes is point `${HOME}/.spacelift/current` to the new location. You can also delete stored credetials for a given profile by using the `spacelift-cli account logout ${MY_ALIAS}` command.

## Usage

Currently only a small subset of operations is supported, starting with the ability to trigger runs and tasks and see their logs.

### Run management

Managing runs is currently avaialble through the `stack` subcommand:

```bash
❯ spacelift-cli stack --id=stack-id help
NAME:
   spacelift-cli stack - Manage a Spacelift stack

USAGE:
   spacelift-cli stack [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command
   Run management:
     deploy   Start a deployment (tracked run)
     logs     Show logs for a particular run
     preview  Start a preview (proposed run)
     task     Perform a task in a workspace

GLOBAL OPTIONS:
   --id value  User-facing ID (slug) of the stack
   --help, -h  show help (default: false)
```

Each of these operations will require user-facing stack ID (so called "slug") visible - among other places - in the stack URL in the GUI. Let's look at the operations one by one.

#### `stack deploy` subcommand

```bash
❯ spacelift-cli stack --id=stack-id deploy --help
NAME:
   spacelift-cli stack deploy - Start a deployment (tracked run)

USAGE:
   spacelift-cli stack deploy [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --sha value  Commit SHA for the newly created run
   --tail       Indicate whether to tail the run (default: false)
   --help, -h   show help (default: false)
```

This subcommand creates a [tracked run](https://docs.spacelift.io/concepts/run/tracked) on a stack. By default, it uses the latest tracked commit, but you can also point it to an arbitrary SHA using the `--sha` flag. Also by default, this command only creates a run and prints out the URL where the run can be accessed. If the `--tail` flag is set however, run logs will be retrieved until the run terminates. If the run is tailed, the exit code of the command will depend on the outcome of the run - if the run is `FINISHED`, the command succeeds (exits with `0`), otherwise it fails.

Notes:

- Terminating the executable (Ctrl+C) while tailing logs does not stop the run, it just stops tailing logs.

- You cannot confirm or discard the run from the command line. If the deployment ends up in [`UNCONFIRMED`](https://docs.spacelift.io/concepts/run/tracked#unconfirmed) state, the CLI be stuck until the state machine progresses. You will need to go to the run URL and make a decision there.

#### `stack preview` subcommand

```bash
❯ spacelift-cli stack --id=stack-id preview --help
NAME:
   spacelift-cli stack preview - Start a preview (proposed run)

USAGE:
   spacelift-cli stack preview [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --sha value  Commit SHA for the newly created run
   --tail       Indicate whether to tail the run (default: false)
   --help, -h   show help (default: false)
```

This subcommand creates a [proposed run](https://docs.spacelift.io/concepts/run/proposed) on a stack. By default, it uses the latest tracked commit, but you can also point it to an arbitrary SHA using the `--sha` flag. Also by default, this command only creates a run and prints out the URL where the run can be accessed. If the `--tail` flag is set however, run logs will be retrieved until the run terminates. If the run is tailed, the exit code of the command will depend on the outcome of the run - if the run is `FINISHED`, the command succeeds (exits with `0`), otherwise it fails.

#### `stack logs` subcommand

```bash
❯ spacelift-cli stack --id=stack-id logs --help
NAME:
   spacelift-cli stack logs - Show logs for a particular run

USAGE:
   spacelift-cli stack logs [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --run value  ID of the run
   --help, -h   show help (default: false)
```

This subcommand prints out (and tails) logs for an existing run or task. The exit code of the command will depend on the outcome of the run - if the run is `FINISHED`, the command succeeds (exits with `0`), otherwise it fails.

Note that `run` is a required parameter here and represents the unique run ID, one that looks like `01F2KB8SARWF3V2PSFYXK5D0S7`.

#### `stack task` subcommand

```bash
❯ spacelift-cli stack --id=stack-id task --help
NAME:
   spacelift-cli stack task - Perform a task in a workspace

USAGE:
   spacelift-cli stack task [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --noinit    Indicate whether to skip initialization for a task (default: false)
   --tail      Indicate whether to tail the run (default: false)
   --help, -h  show help (default: false)
```

This subcommand starts a [task](https://docs.spacelift.io/concepts/run/task) against a stack. The command for the task itself must be specified after all the other `spacelift-cli` command arguments. It can be quoted to prevent any confusion from shell tokenization. Example:

[![asciicast](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B.svg)](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B)

By default, this command only creates a run and prints out the URL where the run can be accessed. If the `--tail` flag is set however (see above), run logs will be retrieved until the run terminates. If the run is tailed, the exit code of the command will depend on the outcome of the run - if the run is `FINISHED`, the command succeeds (exits with `0`), otherwise it fails.
