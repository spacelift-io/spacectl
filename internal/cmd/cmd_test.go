package cmd_test

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/suite"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/nullable"
)

type CommandTests struct {
	suite.Suite
}

func Test_Command(t *testing.T) {
	suite.Run(t, new(CommandTests))
}

func (t *CommandTests) Test_FindLatestSupportedVersion_SaaS() {
	type testCase struct {
		versions        []cmd.VersionedCommand
		expectedVersion cmd.SupportedVersion
	}

	testCases := map[string]testCase{
		"only latest": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: cmd.SupportedVersionLatest,
		},
		"latest and all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: cmd.SupportedVersionLatest,
		},
		"latest and specific versions": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: cmd.SupportedVersionLatest,
		},
		"specific versions": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
			},
			expectedVersion: cmd.SupportedVersion("2.3.3"),
		},
		"only all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
			},
			expectedVersion: cmd.SupportedVersionAll,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func() {
			command := cmd.Command{
				Versions: testCase.versions,
			}

			supportedVersion := command.FindLatestSupportedVersion(cmd.SpaceliftInstanceVersion{
				InstanceType: cmd.SpaceliftInstanceTypeSaaS,
			})

			t.Require().NotNil(supportedVersion)
			t.Equal(testCase.expectedVersion, supportedVersion.EarliestVersion)
		})
	}
}

func (t *CommandTests) Test_FindLatestSupportedVersion_Unknown() {
	type testCase struct {
		versions        []cmd.VersionedCommand
		expectedVersion *cmd.SupportedVersion
	}

	testCases := map[string]testCase{
		"only latest": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: nil,
		},
		"latest and all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
		"latest and specific versions": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			expectedVersion: nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
		"specific versions": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
			},
			expectedVersion: nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
		"only all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
			},
			expectedVersion: nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func() {
			command := cmd.Command{
				Versions: testCase.versions,
			}

			supportedVersion := command.FindLatestSupportedVersion(cmd.SpaceliftInstanceVersion{
				InstanceType: cmd.SpaceliftInstanceTypeUnknown,
			})

			if testCase.expectedVersion != nil {
				t.Require().NotNil(supportedVersion)
				t.Equal(*testCase.expectedVersion, supportedVersion.EarliestVersion)
			} else {
				t.Nil(supportedVersion)
			}
		})
	}
}

func (t *CommandTests) Test_FindLatestSupportedVersion_SelfHosted() {
	type testCase struct {
		versions          []cmd.VersionedCommand
		selfHostedVersion *semver.Version
		expectedVersion   *cmd.SupportedVersion
	}

	testCases := map[string]testCase{
		"only latest": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			selfHostedVersion: semver.MustParse("1.2.0"),
			expectedVersion:   nil,
		},
		"latest and all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			selfHostedVersion: semver.MustParse("1.2.0"),
			expectedVersion:   nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
		"latest and specific versions": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
				{EarliestVersion: cmd.SupportedVersionLatest},
			},
			selfHostedVersion: semver.MustParse("1.5.3"),
			expectedVersion:   nullable.OfValue(cmd.SupportedVersion("1.5.0")),
		},
		"specific versions with version match": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
			},
			selfHostedVersion: semver.MustParse("1.5.3"),
			expectedVersion:   nullable.OfValue(cmd.SupportedVersion("1.5.0")),
		},
		"specific versions with no version match": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("1.5.0")},
				{EarliestVersion: cmd.SupportedVersion("2.3.3")},
			},
			selfHostedVersion: semver.MustParse("1.1.3"),
			expectedVersion:   nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
		"ignores prerelease information in version number": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
				{EarliestVersion: cmd.SupportedVersion("1.2.0")},
				{EarliestVersion: cmd.SupportedVersion("3.0.0")},
			},
			selfHostedVersion: semver.MustParse("3.0.0-alpha.2"),
			expectedVersion:   nullable.OfValue(cmd.SupportedVersion("3.0.0")),
		},
		"only all": {
			versions: []cmd.VersionedCommand{
				{EarliestVersion: cmd.SupportedVersionAll},
			},
			selfHostedVersion: semver.MustParse("1.1.3"),
			expectedVersion:   nullable.OfValue[cmd.SupportedVersion](cmd.SupportedVersionAll),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func() {
			command := cmd.Command{
				Versions: testCase.versions,
			}

			supportedVersion := command.FindLatestSupportedVersion(cmd.SpaceliftInstanceVersion{
				InstanceType: cmd.SpaceliftInstanceTypeSelfHosted,
				Version:      testCase.selfHostedVersion,
			})

			if testCase.expectedVersion != nil {
				t.Require().NotNil(supportedVersion)
				t.Equal(*testCase.expectedVersion, supportedVersion.EarliestVersion)
			} else {
				t.Nil(supportedVersion)
			}
		})
	}
}
