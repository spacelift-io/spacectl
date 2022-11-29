package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

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
	artifactsData, err := os.ReadFile(dir + "/artifacts.json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read artifacts.json")
	}

	if err := json.Unmarshal(artifactsData, &out.Artifacts); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal artifacts.json")
	}

	// Read the metadata file.
	metadataData, err := os.ReadFile(dir + "/metadata.json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read metadata.json")
	}

	if err := json.Unmarshal(metadataData, &out.Metadata); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal metadata.json")
	}

	// Read the CHANGELOG.
	notesData, err := os.ReadFile(dir + "/CHANGELOG.md")
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to read notes.txt")
		}
	} else {
		notes := string(notesData)
		out.Changelog = &notes
	}

	return &out, nil
}

type GoReleaserArtifacts []GoReleaserArtifact

func (a GoReleaserArtifacts) Archives() []GoReleaserArtifact {
	var archives []GoReleaserArtifact

	for _, artifact := range a {
		if artifact.Type == "Archive" {
			archives = append(archives, artifact)
		}
	}

	return archives
}

func (a GoReleaserArtifacts) ChecksumsFile() (*GoReleaserArtifact, error) {
	for _, artifact := range a {
		if artifact.Type == "Checksum" {
			return &artifact, nil
		}
	}

	return nil, errors.New("checksums file not found")
}

func (a GoReleaserArtifacts) SignatureFile() (*GoReleaserArtifact, error) {
	for _, artifact := range a {
		if artifact.Type == "Signature" {
			return &artifact, nil
		}
	}

	return nil, errors.New("signature file not found")
}

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
	content, err := a.Content(dir)
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
func (a *GoReleaserArtifact) Upload(ctx context.Context, dir string, url string) error {
	content, err := a.Content(dir)
	if err != nil {
		return errors.Wrapf(err, "could not get artifact content for %s", a.Name)
	}
	defer content.Close()

	request, err := http.NewRequest(http.MethodPut, url, content)
	if err != nil {
		return errors.Wrapf(err, "could not create request for %s", a.Name)
	}

	response, err := ctxhttp.Do(ctx, http.DefaultClient, request)
	if err != nil {
		return errors.Wrapf(err, "could not upload %s", a.Name)
	}

	if response.StatusCode/100 != 2 {
		return errors.Errorf("could not upload %s: %s", a.Name, response.Status)
	}

	return nil
}

func (a *GoReleaserArtifact) Content(dir string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(dir, a.Path))
}

type GoReleaserArtifactExtras struct {
	Checksum GoReleaserArtifactChecksum `json:"Checksum"`
}

type GoReleaserArtifactChecksum string

// BinarySHA256 returns the binary SHA256 checksum as a hex-encoded string.
func (c GoReleaserArtifactChecksum) BinarySHA256() string {
	return strings.TrimPrefix(string(c), "sha256:")
}

type GoReleaserMetadata struct {
	Version string `json:"version"`
}
