package internal

import (
	"fmt"
	"strings"
)

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

func (v Version) Row() []string {
	return []string{
		v.ID,
		v.Number,
		v.Status,
		v.Platforms.String(),
	}
}

type Versions []Version

func (v Versions) Headers() []string {
	return []string{
		"ID",
		"Number",
		"Status",
		"Platforms",
	}
}

type VersionPlatforms []VersionPlatform

func (p VersionPlatforms) String() string {
	partial := make([]string, len(p))

	for i, platform := range p {
		partial[i] = platform.String()
	}

	return strings.Join(partial, ", ")
}

type VersionPlatform struct {
	Architecture string `graphql:"architecture" json:"architecture"`
	OS           string `graphql:"os" json:"os"`
}

func (p VersionPlatform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Architecture)
}
