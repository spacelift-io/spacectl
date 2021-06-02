# `spacectl`, the Spacelift CLI

`spacectl` is a utility wrapping Spacelift's [GraphQL API](https://docs.spacelift.io/integrations/api) for easy programmatic access in command-line contexts - either in manual interactive mode (in your local shell), or in a predefined CI pipeline (GitHub actions, CircleCI, Jenkins etc).

## Installation

`spacectl` is distributed through GitHub Releases as a zip file containing a self-contained statically linked executable built from the source in this repository. Binaries can be download directly from the [Releases page](https://github.com/spacelift-io/spacectl/releases).

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

- `SPACELIFT_API_ENDPOINT` - the URL to your Spacelift account, for example `https://mycorp.app.spacelift.io`.
- `SPACELIFT_API_GITHUB_TOKEN` - a GitHub personal access token.

#### Spacelift API keys

To use a Spacelift API key, set the following environment variables:

- `SPACELIFT_API_ENDPOINT` - the URL to your Spacelift account, for example `https://mycorp.app.spacelift.io`.
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
   current  Outputs your currently selected profile
   list     List all your Spacelift account profiles
   login    Create a profile for a Spacelift account
   logout   Remove Spacelift credentials for an existing profile
   select   Select one of your Spacelift account profiles
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

Each of the subcommands requires an account **alias**, which is a short, user-friendly name for each set of credentials (account profiles). Profiles don't need to be unique - you can have multiple sets of credentials for a single account too.

Account profiles don't use short-lived tokens, so GitHub access tokens and API keys are the only two supported authentication methods. In order to authenticate to your first profile, type in the following (make sure to replace `${MY_ALIAS}` with the actual profile alias):

```bash
❯ spacectl profile login ${MY_ALIAS}
Enter Spacelift endpoint (eg. https://unicorn.app.spacelift.io/):
```

In the next step, you will be asked to choose which authentication method you are going to use. Note that if your account is using [SAML-based SSO authentication](https://docs.spacelift.io/integrations/single-sign-on), then API keys are your only option. After you're done entering credentials, the CLI will validate them against the server, and assuming that they're valid, will persist them in a credentials file in `.spacelift/${MY_ALIAS}`. It will also create a symlink in `${HOME}/.spacelift/current` pointing to the current profile.

You can switch between account profiles by using `spacectl profile select ${MY_ALIAS}`. What this does behind the scenes is point `${HOME}/.spacelift/current` to the new location. You can also delete stored credetials for a given profile by using the `spacectl profile logout ${MY_ALIAS}` command.

## Usage

Currently only a small subset of operations is supported, starting with the ability to trigger runs and tasks and see their logs.

### Run management

Managing runs is currently avaialble through the `stack` subcommand:

```bash
❯ spacectl stack --id=stack-id help
NAME:
   spacectl stack - Manage a Spacelift stack

USAGE:
   spacectl stack [global options] command [command options] [arguments...]

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
❯ spacectl stack --id=stack-id deploy --help
NAME:
   spacectl stack deploy - Start a deployment (tracked run)

USAGE:
   spacectl stack deploy [command options] [arguments...]

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
❯ spacectl stack --id=stack-id preview --help
NAME:
   spacectl stack preview - Start a preview (proposed run)

USAGE:
   spacectl stack preview [command options] [arguments...]

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
❯ spacectl stack --id=stack-id logs --help
NAME:
   spacectl stack logs - Show logs for a particular run

USAGE:
   spacectl stack logs [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --run value  ID of the run
   --help, -h   show help (default: false)
```

This subcommand prints out (and tails) logs for an existing run or task. Note that for an existing run or task, the `logs` subcommand will always succeed (exit with `0`), regardless of its outcome.

Note that `run` is a required parameter here and represents the unique run ID, one that looks like `01F2KB8SARWF3V2PSFYXK5D0S7`.

#### `stack task` subcommand

```bash
❯ spacectl stack --id=stack-id task --help
NAME:
   spacectl stack task - Perform a task in a workspace

USAGE:
   spacectl stack task [command options] [arguments...]

CATEGORY:
   Run management

OPTIONS:
   --noinit    Indicate whether to skip initialization for a task (default: false)
   --tail      Indicate whether to tail the run (default: false)
   --help, -h  show help (default: false)
```

This subcommand starts a [task](https://docs.spacelift.io/concepts/run/task) against a stack. The command for the task itself must be specified after all the other `spacectl` command arguments. It can be quoted to prevent any confusion from shell tokenization. Example:

[![asciicast](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B.svg)](https://asciinema.org/a/pYm8lqM5XTUoG1UsDo7OL6t8B)

By default, this command only creates a run and prints out the URL where the run can be accessed. If the `--tail` flag is set however (see above), run logs will be retrieved until the run terminates. If the run is tailed, the exit code of the command will depend on the outcome of the run - if the run is `FINISHED`, the command succeeds (exits with `0`), otherwise it fails.
