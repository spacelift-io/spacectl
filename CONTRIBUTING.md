# spacectl Contributing Guide

In this guide you will get an overview of the contribution workflow from opening an issue, creating a pull request, to having it reviewed, merged and deployed.

We welcome all contributions, no matter the size or complexity. Every contribution helps and is appreciated.

The following is a set of guidelines for contributing to spacectl. These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

## Reporting an Issue

If you spot a problem with spacectl [search if an issue already exists](https://github.com/spacelift-io/spacectl/issues). If a related issue doesn't exist, please open a new issue using [the issue form](https://github.com/spacelift-io/user-documentation/issues/new).

## Making Changes

spacectl is written in Go, and uses the [Spacelift GraphQL API](https://docs.spacelift.io/integrations/api) to interact with Spacelift. To make changes to spacectl you will need at least the following:

- A working copy of [Go](https://go.dev/). Check [go.mod](./go.mod) to see what is the current version used by this project.
- A Spacelift account you can use for testing. Please go to <https://spacelift.io/free-trial> to create a free account if you don't already have one.

### Command versioning

spacectl supports working with both SaaS and Self-Hosted instances of Spacelift. Because of this we need to have a mechanism in place for ensuring API compatibility with different versions. To support this, each command can have multiple different implementations, which work with different versions of Spacelift.

#### Defining a command

Commands are defined using the `cmd.Command` struct like in the following example:

```go
{
	Category: "Run local preview",
	Name:     "local-preview",
	Usage:    "Start a preview (proposed run) based on the current project. Respects .gitignore and .terraformignore.",
	Versions: []cmd.VersionedCommand{
		{
			EarliestVersion: cmd.SupportedVersionAll,
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(false),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
		{
			EarliestVersion: cmd.SupportedVersion("2.5.0"),
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(true),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	},
},
```

In the example above, we are defining a new subcommand for the `stack` command called `local-preview`. There are two versions of the command: one that works with any version of Spacelift, and another that only works with versions 2.5.0 and above.

The `EarliestVersion` field can have the following values:

- `SupportedVersionAll` - indicating it can be used with any version of Spacelift.
- `SupportedVersionLatest` - indicating it can only be used with Spacelift's SaaS product, but will be available in the next Self-Hosted version.
- A specific version number, for example `2.5.0` - indicating that it can be used with Spacelift SaaS, or a Self-Hosted instance running v2.5.0 or above.

#### Adding a new command

When adding a new command the safest thing to do is set the `EarliestVersion` to `SupportedVersionLatest`. This means that it will only be available to Self-Hosted installations after the next release of Self-Hosted.

The exception to this is if you are certain that the API your command needs is available in all supported Self-Hosted versions, in which case you can use `SupportedVersionAll`.

#### Making changes to existing commands

When making changes to existing spacectl commands, if your change relies on a GraphQL API feature that isn't currently released to all supported versions of Self-Hosted, please add a new version of the command with the `EarliestVersion` set to `SupportedVersionLatest`.

## Submitting Changes

Once you are happy with your changes, just open a pull request.

By submitting a pull request for this project, you agree to license your contribution under the [MIT license](./LICENSE) to Spacelift.

## Releasing

### Self-Hosted Releases

If a new version of Self-Hosted has been released, and you want to enable functionality that was previously waiting for the changes to become available in the Self-Hosted GraphQL API, find any references to `cmd.SupportedVersionLatest`, and replace them with the new Self-Hosted version.

For example, if we have the following command definition:

```go
{
	Category: "Run local preview",
	Name:     "local-preview",
	Usage:    "Start a preview (proposed run) based on the current project. Respects .gitignore and .terraformignore.",
	Versions: []cmd.VersionedCommand{
		{
			EarliestVersion: cmd.SupportedVersionAll,
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(false),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
		{
			EarliestVersion: cmd.SupportedVersionLatest,
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(true),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	},
},
```

And we have just released version 2.5.0 of Self-Hosted with support for the new API functionality needed by the command, we would change it to the following:

```go
{
	Category: "Run local preview",
	Name:     "local-preview",
	Usage:    "Start a preview (proposed run) based on the current project. Respects .gitignore and .terraformignore.",
	Versions: []cmd.VersionedCommand{
		{
			EarliestVersion: cmd.SupportedVersionAll,
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(false),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
		{
			EarliestVersion: cmd.SupportedVersion("2.5.0"), // <-- Latest replaced by 2.5.0.
			Command: &cli.Command{
				Flags: []cli.Flag{
					flagStackID,
					// ... other flags
				},
				Action:    localPreview(true),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	},
},
```

### Releasing spacectl

To release a new version of spacectl, tag the repo and then push that tag. For example:

```shell
git checkout main
git pull
git tag -a v0.24.2 -m"Releasing v0.24.2"
git push origin v0.24.2
```

After the tag is pushed, the [release](.github/workflows/release.yml) workflow will trigger and will automatically publish a new GitHub release.

Once the [release](.github/workflows/release.yml) workflow is done, go to the [winget-pkgs](https://github.com/microsoft/winget-pkgs) repository and submit a pull request for the release version.
