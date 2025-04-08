package cmd

import (
	"slices"

	"github.com/Masterminds/semver/v3"
	"github.com/urfave/cli/v2"

	spslices "github.com/spacelift-io/spacectl/internal/slices"
)

func ResolveCommands(instanceVersion SpaceliftInstanceVersion, allCommands []Command) []*cli.Command {
	var availableCommands []*cli.Command
	for _, command := range allCommands {
		latestVersion := command.FindLatestSupportedVersion(instanceVersion)
		if latestVersion != nil {
			latestVersion.Command.Name = command.Name
			latestVersion.Command.Usage = command.Usage
			latestVersion.Command.Category = command.Category

			// TODO: make this recursive to ensure we get all levels of the command tree
			var subcommands []*cli.Command
			for _, subcommand := range command.Subcommands {
				latestVersion := subcommand.FindLatestSupportedVersion(instanceVersion)
				if latestVersion != nil {
					latestVersion.Command.Name = subcommand.Name
					latestVersion.Command.Usage = subcommand.Usage
					latestVersion.Command.Category = subcommand.Category

					subcommands = append(subcommands, latestVersion.Command)
				}
			}

			latestVersion.Command.Subcommands = subcommands

			availableCommands = append(availableCommands, latestVersion.Command)
		}
	}

	return availableCommands
}

// SupportedVersion is used to indicate what versions of Spacelift certain spacectl commands are
// compatible with.
type SupportedVersion string

const (
	// SupportedVersionAll indicates that the command is available for all versions of Spacelift.
	SupportedVersionAll = "all"

	// SupportedVersionLatest indicates that the command is supported for SaaS, and will be supported
	// by Self-Hosted after the next Self-Hosted release.
	SupportedVersionLatest = "latest"
)

type Command struct {
	Name     string
	Usage    string
	Category string

	// Versions defines the available versions for the command.
	Versions []VersionedCommand

	// Subcommands gets the list of subcommands that support the specified version.
	Subcommands []Command
}

type VersionedCommand struct {
	// EarliestVersion indicates that the command needs at least the indicated Self-Hosted version
	// in order to work.
	//
	// - SupportedVersionAll - indicates that the command can be used for any Spacelift version (both SaaS and Self-Hosted).
	// - SupportedVersionLatest - indicates that the command can be used with SaaS, but will not be available to Self-Hosted until the next release.
	// - 1.2.3, 2.5.0, etc - indicates that the command can be used with SaaS, or a Self-Hosted version equal to or higher than the specified version.
	EarliestVersion SupportedVersion

	// The CLI command definition.
	Command *cli.Command
}

type SpaceliftInstanceVersion struct {
	// SaaS indicates that we are communicating with a Spacelift SaaS instance.
	SaaS bool

	// Version indicates the Self-Hosted version we are communicating with. It will be nil for SaaS.
	Version *semver.Version
}

// FindLatestSupportedVersion finds the latest supported version of the specified command. It returns
// nil if no version of the command is supported by the Spacelift instance.
func (c Command) FindLatestSupportedVersion(instanceVersion SpaceliftInstanceVersion) *VersionedCommand {
	// // Get potential list of commands:
	// //   - If not SaaS, we can remove the "latest" version.
	// //   - If self-hosted we can remove any versions that are not supported by the instance.
	// // Sort the commands into order of preference: "latest", specific versions, then "all".
	// // Return the first command, or nil if none is supported.
	availableCommands := c.Versions
	if !instanceVersion.SaaS {
		availableCommands = spslices.Filter(availableCommands, func(v VersionedCommand) bool {
			if v.EarliestVersion == SupportedVersionLatest {
				// If the version is marked as "latest", it won't be available yet in a Self-Hosted
				// instance because a version of Self-Hosted supporting the command hasn't been released
				// yet.
				return false
			}

			if v.EarliestVersion == SupportedVersionAll {
				// This command is supported for all versions of Spacelift.
				return true
			}

			// We're assuming that because the command versions are defined statically in the code,
			// the version should always parse.
			version := semver.MustParse(string(v.EarliestVersion))

			// If the command only supports certain Self-Hosted instances, only include it if
			// the Spacelift instance we're connecting to is running at least the earliest version
			// supported by the command.
			return instanceVersion.Version.GreaterThanEqual(version)
		})
	}

	if len(availableCommands) == 0 {
		return nil
	}

	slices.SortStableFunc(availableCommands, func(a, b VersionedCommand) int {
		if a.EarliestVersion == b.EarliestVersion {
			return 0
		}

		if a.EarliestVersion == SupportedVersionLatest {
			return -1
		}

		if b.EarliestVersion == SupportedVersionLatest {
			return 1
		}

		if a.EarliestVersion == SupportedVersionAll {
			return 1
		}

		if b.EarliestVersion == SupportedVersionAll {
			return -1
		}

		aVersion := semver.MustParse(string(a.EarliestVersion))
		bVersion := semver.MustParse(string(b.EarliestVersion))

		// Otherwise we sort in semver order with the latest version coming first.
		return bVersion.Compare(aVersion)
	})

	return &availableCommands[0]
}
