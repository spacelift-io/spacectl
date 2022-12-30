package internal

import (
	"fmt"
	"strings"
)

// Version is a version of a Terraform provider.
type Version struct {
	ID               string           `graphql:"id" json:"id"`
	CreatedAt        int64            `graphql:"createdAt" json:"createdAt"`
	Description      *string          `graphql:"description" json:"description"`
	Number           string           `graphql:"number" json:"number"`
	Platforms        VersionPlatforms `graphql:"platforms" json:"platforms"`
	ProtocolVersions []string         `graphql:"protocolVersions" json:"protocolVersions"`
	Status           string           `graphql:"status" json:"status"`
	UpdatedAt        int64            `graphql:"updatedAt" json:"updatedAt"`
}

// Row returns a slice of strings representing a row in a table of provider
// versions.
func (v Version) Row() []string {
	return []string{
		v.ID,
		v.Number,
		v.Status,
		v.Platforms.String(),
	}
}

// Versions is a slice of provider versions.
type Versions []Version

// Headers returns a collection of versions table headers.
func (v Versions) Headers() []string {
	return []string{
		"ID",
		"Number",
		"Status",
		"Platforms",
	}
}

// VersionPlatforms is a slice of provider version platforms.
type VersionPlatforms []VersionPlatform

// String returns a comma-separated list of platforms.
func (p VersionPlatforms) String() string {
	partial := make([]string, len(p))

	for i, platform := range p {
		partial[i] = platform.String()
	}

	return strings.Join(partial, ", ")
}

// VersionPlatform is a platform for a provider version.
type VersionPlatform struct {
	Architecture string `graphql:"architecture" json:"architecture"`
	OS           string `graphql:"os" json:"os"`
}

// String returns a string representation of a platform.
func (p VersionPlatform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Architecture)
}
