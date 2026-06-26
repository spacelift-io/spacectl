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

// Headers returns the table headers matching Row.
func (v Version) Headers() []string {
	return []string{
		"ID",
		"Number",
		"Status",
		"Platforms",
	}
}

// VerifiedVersion is a provider version that also exposes the verification-related fields.
type VerifiedVersion struct {
	Version
	VerifiedAt          *int64  `graphql:"verifiedAt" json:"verifiedAt"`
	VerificationFailure *string `graphql:"verificationFailure" json:"verificationFailure"`
}

func (vv VerifiedVersion) verificationStatus() string {
	if vv.VerificationFailure != nil {
		return "failed"
	}

	if vv.VerifiedAt != nil {
		return "valid"
	}

	return "pending"
}

// Row returns a slice of strings representing a row in a table of provider
// versions, including the verification status.
func (vv VerifiedVersion) Row() []string {
	return []string{
		vv.ID,
		vv.Number,
		vv.Status,
		vv.verificationStatus(),
		vv.Platforms.String(),
	}
}

// Headers returns the table headers matching Row.
func (vv VerifiedVersion) Headers() []string {
	return []string{
		"ID",
		"Number",
		"Status",
		"Verified",
		"Platforms",
	}
}

// Versioned is implemented by the provider version types that can be
// rendered as a table, with or without the verification-related fields.
type Versioned interface {
	Version | VerifiedVersion
	Row() []string
	Headers() []string
}

// Versions is a slice of provider versions.
type Versions[V Versioned] []V

// Headers returns a collection of versions table headers.
func (Versions[V]) Headers() []string {
	var version V
	return version.Headers()
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
