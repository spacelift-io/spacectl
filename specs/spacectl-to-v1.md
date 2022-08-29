# Getting spacectl to v1

This document describes the work that needs to be done in order to get spacectl to the point where it’s stable and allows users to perform most of the tasks that are available via the web UI. Its other aim is to make sure we plan the structure of spacectl’s commands, to provide a consistent interface, and to minimize the amount of deprecations we need to make during development.

The list of commands aren't meant to be set in stone, and we might find we need to change our
plans as we implement functionality, but at least this gives us a chance to think about the
design up-front, and aim for a consistent experience.

## Project Tasks

There are a number of tasks that need to be completed that aren’t specifically related to work on the codebase itself, but that are important so that we can encourage people to use spacectl, and to contribute to the project. These include:

- Adding a contribution guide.
- Creating PR and issue templates.
- Deciding how we handle triage for the project.
- Adding documentation to docs.spacelift.io.

## Functionality

### Command Arguments

At the moment we have a situation where flags need to be specified in multiple places. This leads to a slightly strange situation where to view the help for the `set-current-commit` command you need to provide the `--id` argument, as reported in [#19](https://github.com/spacelift-io/spacectl/issues/19):

```shell
spacectl stack --id placeholder set-current-commit --help
```

This is unintuitive, and the following would be more common:

```shell
spacectl stack set-current-commit --help
```

We should make sure that all commands use the following format for consistency:

```shell
spacectl <command> [<subcommand>...] [options...] [arguments...]
```

### Output Formats

There are a number of different use-cases for CLI tools, including users manually running commands for one-off tasks or to explore functionality, as well as for automation via scripting. A different view of the data is required depending on the use-case. For example, a user might want the results to be formatted in a table, but in a script you might want to get the output as JSON in order to process it automatically.

Because of this, all commands that produce some kind of non-interactive output should support an output format flag:

```shell
-o|--output-format [table|json]
```

#### Supported Formats

At a minimum we should support `table` and `json` formats. This provides a human-readable format
(`table`) for interactive usage, as well as a machine-readable format for scripting (`json`).
We can add new formats later, for example yaml, if users request them.

#### Default Format

The default format should be `table` so that users don’t have to specify that every time they enter a command manually.

### Logging

Because one of the use-cases is scripting, we want to be able to use the output of spacectl in redirects and pipelines without log messages ending up in the output. To support this, we could write any log messages to `stderr`, and keep `stdout` for content.

### Commands

#### API Keys

- `spacectl api-key`
  - [ ] `list` - lists the API keys in your account
  - [ ] `create` - creates an API key
  - [ ] `delete` - deletes an API key

#### Audit Trail

- `spacectl audit-trail`
  - [ ] `show` - shows current audit trail settings
  - [ ] `setup` - configures the audit trail
  - [ ] `update` - allows you to update the webhooks endpoint, secret and status of the integration

#### Modules

We should support the following commands for working with Modules:

- `spacectl module`
  - [ ] `list` - lists the modules you have access to
  - [ ] `add` - adds a new module
  - [ ] `delete` - deletes a module
  - [ ] `show` - outputs information about a specified module
  - [ ] `edit` - edits the name, provider, labels and description for the module

Subcommands:

- `spacectl module version`
  - [ ] `list` - lists the versions for a specific module
  - [ ] `create` - triggers a run to create a new version
  - [ ] `show` - shows information about a module version
- `spacectl module pr`
  - [ ] `list` - lists any open PRs against the module
  - [ ] `show` - outputs information about a specified PR
- `spacectl module environment`
  - [ ] `show` - outputs the environment variables and attached contexts.
  - [ ] `setvar` - sets an environment variable
  - [ ] `mount` - mounts a file from existing file or STDIN
  - [ ] `delete` - deletes an environment variable or mounted file
- `spacectl module vcs`
  - [ ] `show` - show the module's VCS settings
  - [ ] `edit` - edit the module's VCS settings
- `spacectl module behavior`
  - [ ] `show` - shows the module's behavior settings
  - [ ] `edit` - edits the module's behavior settings
- `spacectl module context`
  - [ ] `list` - lists the contexts attached to the module
  - [ ] `attach` - attaches a context to the module
  - [ ] `detach` - detaches a context from the module
- `spacectl module policy`
  - [ ] `list` - lists the policies attached to the module
  - [ ] `attach` - attaches a policy to the module
  - [ ] `detach` - detaches a policy from the module
- `spacectl module share`
  - [ ] `list` - lists the accounts that the module is shared with
  - [ ] `add` - adds an account to the list of shared accounts
  - [ ] `remove` - removes an account from the list of shared accounts

#### Policies

We should support the following commands for working with policies:

- `spacectl policy`
  - [ ] `list` - lists the policies in the account
  - [ ] `show` - shows the details of a specific policy
  - [ ] `add` - adds a new policy
  - [ ] `edit` - updated an existing policy
  - [ ] `delete` - deletes a policy

#### Profile

- `spacectl profile`
  - [ ] `current` - shows the currently active profile
  - [ ] `list` - lists available profiles
  - [ ] `login` - creates a new profile
  - [ ] `logout` - removes a profile
  - [ ] `select` - selects one of your profiles as the current profile

#### Slack Settings

**_Question_**: maybe this is overkill? There isn't really much you can do with the Slack
settings other than viewing that it's configured and opening a browser to connect.

- `spacectl slack`
  - [ ] `show` - shows current integration settings
  - [ ] `setup` - configures the Slack integration by launching the OAuth2 process

#### SSO Settings

We should support the following commands for working with SSO settings.

- `spacectl sso`
  - [ ] `show` - show current SSO settings
  - [ ] `setup` - configure the SSO settings for the account
  - [ ] `delete` - deletes your SSO configuration

#### Stacks

We should support the following commands for working with Stacks:

- `spacectl stack`
  - [x] `list` - lists the stacks you have access to
  - [ ] `add` - adds a new stack
  - [ ] `delete` - deletes a stack
  - [x] `show` - outputs information about a specified Stack
  - [ ] `edit` - edits the name, labels and description for the stack
  - [x] `set-current-commit` - sets the current commit for the stack
  - [x] `confirm` - confirms a run awaiting approval
  - [x] `discard` - discards a run awaiting approval
  - [x] `approve` - approves a run or task
  - [x] `reject` - rejects a run or task
  - [x] `deploy` - triggers a tracked (i.e. deployment) run
  - [x] `preview` - triggers a preview run for a specific commit
  - [x] `local-preview` - triggers a local-preview run using the current directory as the workspace
  - [x] `task` - performs a one-off task in a workspace

Subcommands:

- `spacectl stack run`
  - [x] `list` - lists the runs for your stack
  - [ ] `logs` - shows the logs for a run
  - [ ] `show` - outputs information about a specified Run
- `spacectl stack task`
  - [ ] `list` - lists any tasks that have been run against the stack
  - [ ] `show` - outputs information about a specified Task
- `spacectl stack pr`
  - [ ] `list` - lists any open PRs against the stack
  - [ ] `show` - outputs information about a specified PR
- `spacectl stack environment`
  - [x] `list` - outputs the environment variables and attached contexts.
  - [x] `setvar` - sets an environment variable
  - [x] `mount` - mounts a file from existing file or STDIN
  - [x] `delete` - deletes an environment variable or mounted file

The following commands are all under the _settings_ section of the UI, but they aren't nested
under a `settings` subcommand here to avoid too much command nesting. Alternatively we could
just have a single settings command that does everything:

- `spacectl stack vcs`
  - [ ] `show` - show the stack's VCS settings
  - [ ] `edit` - edit the stack's VCS settings
- `spacectl stack backend`
  - [ ] `show` - shows the stack's backend settings
  - [ ] `edit` - edits the stack's backend settings
- `spacectl stack behavior`
  - [ ] `show` - shows the stack's behavior settings
  - [ ] `edit` - edits the stack's behavior settings
- `spacectl stack context`
  - [ ] `list` - lists the contexts attached to the stack
  - [ ] `attach` - attaches a context to the stack
  - [ ] `detach` - detaches a context from the stack
- `spacectl stack policy`
  - [ ] `list` - lists the policies attached to the stack
  - [ ] `attach` - attaches a policy to the stack
  - [ ] `detach` - detaches a policy from the stack

**_Questions_**:

- Do we want to have any commands for getting stack resources, or is that overkill?
- Is it worth adding commands for managing the integrations (AWS, GCP), or does that not make sense?

#### VCS Agent Pools

We should support the following commands for managing VCS agent pools:

- `spacectl vcs-agent-pool`
  - [ ] `list` - lists the VCS agent pools in the account
  - [ ] `add` - adds a new VCS agent pool
  - [ ] `show` - shows details of a specific agent pool
  - [ ] `reset` - resets the credentials for an agent pool
  - [ ] `delete` - deletes an agent pool

#### VCS Integrations

We should support the following commands for managing VCS integrations.

- `spacectl bitbucket cloud`
  - [ ] `show` - show current integration settings
  - [ ] `setup` - configures the Bitbucket integration
  - [ ] `update` - allows you to update your username and password
  - [ ] `unlink` - disables the integration and removes any settings
- `spacectl bitbucket datacenter`
  - [ ] `show` - show current integration settings
  - [ ] `setup` - configures the Bitbucket integration
  - [ ] `update` - allows you to update your API Host, user facing host, token or webhook secret
  - [ ] `unlink` - disables the integration and removes any settings
- `spacectl github-enterprise`
  - [ ] `show` - show current integration settings
  - [ ] `setup` - configures the GitHub Enterprise integration
  - [ ] `update` - allows you to update your API Host, private key or webhook secret
  - [ ] `unlink` - disables the integration and removes any settings
- `spacectl gitlab`
  - [ ] `show` - show current integration settings
  - [ ] `setup` - configures the GitLab integration
  - [ ] `update` - allows you to update your API Host, token or webhook secret
  - [ ] `unlink` - disables the integration and removes any settings

**_Questions_**:

- Would it make more sense to have a `vcs-provider` command rather than a
  separate command for each integration? It would need to be able to handle the slightly
  different settings that each integration type needed, but it'd mean you could just do
  `spacectl vcs-provider list` to list all the configured integrations, for example.

#### Worker Pools

We should support the following commands for managing worker pools:

- `spacectl worker-pool`
  - [ ] `list` - lists the worker pools in the account
  - [ ] `add` - adds a new worker pool - `spacectl` could take care of the CSR part automatically
  - [ ] `show` - shows details of a specific worker pool
  - [ ] `reset` - resets the credentials for a worker pool
  - [ ] `delete` - deletes a worker pool
