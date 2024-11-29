package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// GoReleaserVersionData contains the data we get from GoReleaser's distribution
// directory.
type GoReleaserVersionData struct {
	Artifacts GoReleaserArtifacts
	Metadata  GoReleaserMetadata
	Changelog *string
}

// BuildGoReleaserVersionData builds a GoReleaserMetadata struct by looking at the
// distribution directory created by GoReleaser.
func BuildGoReleaserVersionData(dir string) (*GoReleaserVersionData, error) {
	var out GoReleaserVersionData

	// Read the artifacts file.
	artifactsPath := filepath.Join(dir, "/artifacts.json")

	// #nosec G304
	artifactsData, err := os.ReadFile(artifactsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", artifactsPath)
	}

	if err := json.Unmarshal(artifactsData, &out.Artifacts); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s", artifactsPath)
	}

	// Read the metadata file.
	metadataPath := filepath.Join(dir, "/metadata.json")

	// #nosec G304
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", metadataPath)
	}

	if err := json.Unmarshal(metadataData, &out.Metadata); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s", metadataPath)
	}

	// Read the CHANGELOG.
	changelogPath := filepath.Join(dir, "/CHANGELOG.md")

	// #nosec G304
	notesData, err := os.ReadFile(changelogPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "failed to read changelog: %s", changelogPath)
		}
	} else {
		notes := string(notesData)
		out.Changelog = &notes
	}

	return &out, nil
}

// GoReleaserArtifacts is a list of GoReleaser artifacts.
type GoReleaserArtifacts []GoReleaserArtifact

// Archives lists all the zip archives in the GoReleaser artifacts.
func (a GoReleaserArtifacts) Archives() []GoReleaserArtifact {
	var archives []GoReleaserArtifact

	for _, artifact := range a {
		if artifact.Type == "Archive" {
			archives = append(archives, artifact)
		}
	}

	return archives
}

// ChecksumsFile finds a checksums file in the GoReleaser artifacts.
func (a GoReleaserArtifacts) ChecksumsFile() (*GoReleaserArtifact, error) {
	for _, artifact := range a {
		if artifact.Type == "Checksum" {
			return &artifact, nil
		}
	}

	return nil, errors.New("checksums file not found")
}

// SignatureFile finds a signature file in the GoReleaser artifacts.
func (a GoReleaserArtifacts) SignatureFile() (*GoReleaserArtifact, error) {
	for _, artifact := range a {
		if artifact.Type == "Signature" {
			return &artifact, nil
		}
	}

	return nil, errors.New("signature file not found")
}

// GoReleaserArtifact represents a single GoReleaser artifact.
type GoReleaserArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`

	// Architecture data only makes sense for binaries and archives.
	OS   *string `json:"goos"`
	Arch *string `json:"goarch"`

	Extra GoReleaserArtifactExtras `json:"extra"`
}

// Checksum returns the SHA256 checksum of the artifact as a hex-encoded string.
func (a *GoReleaserArtifact) Checksum(dir string) (string, error) {
	content, err := a.content(dir)
	if err != nil {
		return "", err
	}
	defer content.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, content); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Upload uploads the artifact's content to the given URL using HTTP PUT method.
func (a *GoReleaserArtifact) Upload(ctx context.Context, dir string, url string, header http.Header) error {
	content, err := a.content(dir)
	if err != nil {
		return errors.Wrapf(err, "could not get artifact content for %s", a.Name)
	}
	defer content.Close()

	data, err := io.ReadAll(content)
	if err != nil {
		return errors.Wrapf(err, "could not read artifact content for %s", a.Name)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrapf(err, "could not create request for %s", a.Name)
	}

	for k := range header {
		request.Header.Set(k, header.Get(k))
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "could not upload %s", a.Name)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Wrapf(err, "could not read response body for %s", a.Name)
	}

	if response.StatusCode/100 != 2 {
		return errors.Errorf("could not upload %s: %s (BODY %s, URL %s)", a.Name, response.Status, body, url)
	}

	return nil
}

// AWSMetadataHeaders returns the headers required for uploading with an AWS presigned URL.
// Deprecated: Use UploadHeaders from the gql TerraformProviderVersionRegisterPlatformV2 response instead.
func (a *GoReleaserArtifact) AWSMetadataHeaders() http.Header {
	headers := http.Header{}
	if a.OS != nil {
		headers.Set("x-amz-meta-binary-os", *a.OS)
	}

	if a.Arch != nil {
		headers.Set("x-amz-meta-binary-architecture", *a.Arch)
	}

	if checksum := a.Extra.Checksum.BinarySHA256(); checksum != "" {
		headers.Set("x-amz-meta-binary-checksum", checksum)
	}
	return headers
}

// ValidateFilename validates that the artifact's name matches the expected
// format.
func (a *GoReleaserArtifact) ValidateFilename(providerType, versionNumber string) error {
	if a.OS == nil {
		return errors.Errorf("missing OS for %s", a.Name)
	}

	if a.Arch == nil {
		return errors.Errorf("missing architecture for %s", a.Name)
	}

	expectedName := fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", providerType, versionNumber, *a.OS, *a.Arch)

	if a.Name != expectedName {
		return errors.Errorf("unexpected artifact name: %s (expected %s)", a.Name, expectedName)
	}

	return nil
}

func (a *GoReleaserArtifact) content(dir string) (io.ReadCloser, error) {
	path := filepath.Join(dir, a.Name)

	// #nosec G304
	out, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open %s", a.Path)
	}

	return out, nil
}

// GoReleaserArtifactExtras contains extra data about a GoReleaser artifact.
type GoReleaserArtifactExtras struct {
	Checksum GoReleaserArtifactChecksum `json:"Checksum"`
}

// GoReleaserArtifactChecksum is a checksum of a GoReleaser artifact.
type GoReleaserArtifactChecksum string

// BinarySHA256 returns the binary SHA256 checksum as a hex-encoded string.
func (c GoReleaserArtifactChecksum) BinarySHA256() string {
	return strings.TrimPrefix(string(c), "sha256:")
}

// GoReleaserMetadata contains metadata about a GoReleaser release that is
// relevant to Spacelift.
type GoReleaserMetadata struct {
	Version string `json:"version"`
}
