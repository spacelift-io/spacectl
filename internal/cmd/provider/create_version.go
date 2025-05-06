package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func createVersion(useHeadersFromAPI bool) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		// Assuming that spacectl is ran from the root of the repository,
		// containing the release artifacts in the "dist" directory.
		dir := cliCtx.String(flagGoReleaserDir.Name)

		providerType := cliCtx.String(flagProviderType.Name)

		fmt.Println("Retrieving release data from ", dir)
		versionData, err := internal.BuildGoReleaserVersionData(dir)
		if err != nil {
			return errors.Wrap(err, "invalid release data")
		}

		fmt.Println("Creating version ", versionData.Metadata.Version)

		checksumsFile, err := versionData.Artifacts.ChecksumsFile()
		if err != nil {
			return err
		}

		checksumsFileChecksum, err := checksumsFile.Checksum(dir)
		if err != nil {
			return errors.Wrap(err, "could not calculate checksum of checksums file")
		}

		signatureFile, err := versionData.Artifacts.SignatureFile()
		if err != nil {
			return err
		}

		signatureFileChecksum, err := signatureFile.Checksum(dir)
		if err != nil {
			return errors.Wrap(err, "could not calculate checksum of signature file")
		}

		variables := map[string]any{
			"provider": graphql.ID(providerType),
			"input": TerraformProviderVersionInput{
				Number:           versionData.Metadata.Version,
				ProtocolVersions: cliCtx.StringSlice(flagProviderVersionProtocols.Name),
				SHASumsFileSHA:   checksumsFileChecksum,
				SignatureFileSHA: signatureFileChecksum,
				SigningKeyID:     cliCtx.String(flagGPGKeyID.Name),
			},
		}

		var sha256SumsUploadURL, sha256SumsSigUploadURL string
		var sha256SumsUploadHeaders, sha256SumsSigUploadHeaders http.Header
		var versionID string

		// We only introduced the upload headers to the GraphQL API for Self-Hosted v3, so we need to use
		// a fallback in case spacectl is running against older versions.
		if useHeadersFromAPI {
			var createMutation struct {
				CreateTerraformProviderVersion struct {
					SHA256SumsUploadURL     string            `graphql:"sha256SumsUploadURL"`
					SHA256SumsUploadHeaders structs.StringMap `graphql:"sha256SumsUploadHeaders"`

					SHA256SumsSigUploadURL     string            `graphql:"sha256SumsSigUploadURL"`
					SHA256SumsSigUploadHeaders structs.StringMap `graphql:"sha256SumsSigUploadHeaders"`
					Version                    struct {
						ID string `graphql:"id"`
					} `graphql:"version"`
				} `graphql:"terraformProviderVersionCreate(provider: $provider, input: $input)"`
			}

			if err := authenticated.Client.Mutate(cliCtx.Context, &createMutation, variables); err != nil {
				return err
			}

			sha256SumsUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL
			sha256SumsUploadHeaders = createMutation.CreateTerraformProviderVersion.SHA256SumsUploadHeaders.HTTPHeaders()

			sha256SumsSigUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL
			sha256SumsSigUploadHeaders = createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadHeaders.HTTPHeaders()

			versionID = createMutation.CreateTerraformProviderVersion.Version.ID
		} else {
			var createMutation struct {
				CreateTerraformProviderVersion struct {
					SHA256SumsUploadURL    string `graphql:"sha256SumsUploadURL"`
					SHA256SumsSigUploadURL string `graphql:"sha256SumsSigUploadURL"`
					Version                struct {
						ID string `graphql:"id"`
					} `graphql:"version"`
				} `graphql:"terraformProviderVersionCreate(provider: $provider, input: $input)"`
			}

			if err := authenticated.Client.Mutate(cliCtx.Context, &createMutation, variables); err != nil {
				return err
			}

			sha256SumsUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL
			sha256SumsUploadHeaders = checksumsFile.AWSMetadataHeaders()

			sha256SumsSigUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL
			sha256SumsSigUploadHeaders = signatureFile.AWSMetadataHeaders()

			versionID = createMutation.CreateTerraformProviderVersion.Version.ID
		}

		fmt.Println("Uploading the checksums file")
		if err := checksumsFile.Upload(cliCtx.Context, dir, sha256SumsUploadURL, sha256SumsUploadHeaders); err != nil {
			return errors.Wrap(err, "could not upload checksums file")
		}

		fmt.Println("Uploading the signatures file")
		if err := signatureFile.Upload(cliCtx.Context, dir, sha256SumsSigUploadURL, sha256SumsSigUploadHeaders); err != nil {
			return errors.Wrap(err, "could not upload signature file")
		}

		archives := versionData.Artifacts.Archives()
		for i := range archives {
			if err := archives[i].ValidateFilename(providerType, versionData.Metadata.Version); err != nil {
				return errors.Wrapf(err, "invalid artifact filename: %s", archives[i].Name)
			}

			if useHeadersFromAPI {
				if err := registerPlatformV2(cliCtx.Context, dir, versionID, &archives[i]); err != nil {
					return err
				}
			} else {
				if err := registerPlatform(cliCtx.Context, dir, versionID, &archives[i]); err != nil {
					return err
				}
			}
		}
		fmt.Printf("Draft version %s created\n", versionID)

		if versionData.Changelog == nil {
			return nil
		}

		var changelogMutation struct {
			Version struct {
				ID string `graphql:"id"`
			} `graphql:"terraformProviderVersionUpdate(version: $version, description: $description)"`
		}

		variables = map[string]any{
			"version":     graphql.ID(versionID),
			"description": graphql.String(*versionData.Changelog),
		}

		fmt.Println("Uploading the changelog")

		if err := authenticated.Client.Mutate(cliCtx.Context, &changelogMutation, variables); err != nil {
			return errors.Wrap(err, "could not update changelog")
		}

		return nil
	}
}

// deprecated, use registerPlatformV2 instead.
func registerPlatform(ctx context.Context, dir string, versionID string, artifact *internal.GoReleaserArtifact) error {
	var mutation struct {
		RegisterTerraformProviderVersionPlatform string `graphql:"terraformProviderVersionRegisterPlatform(version: $version, input: $input)"`
	}

	archiveChecksum, err := artifact.Checksum(dir)
	if err != nil {
		return errors.Wrap(err, "could not calculate checksum of artifact")
	}

	fmt.Printf("Uploading the artifact for %s/%s\n", *artifact.OS, *artifact.Arch)

	variables := map[string]any{
		"version": graphql.ID(versionID),
		"input": TerraformProviderVersionPlatformInput{
			Architecture:    *artifact.Arch,
			OS:              *artifact.OS,
			ArchiveChecksum: archiveChecksum,
			BinaryChecksum:  artifact.Extra.Checksum.BinarySHA256(),
		},
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	if err := artifact.Upload(ctx, dir, mutation.RegisterTerraformProviderVersionPlatform, artifact.AWSMetadataHeaders()); err != nil {
		return errors.Wrapf(err, "could not upload artifact: %s", artifact.Name)
	}

	return nil
}

func registerPlatformV2(ctx context.Context, dir string, versionID string, artifact *internal.GoReleaserArtifact) error {
	var mutation struct {
		RegisterTerraformProviderVersionPlatform struct {
			UploadURL     string `json:"uploadUrl"`
			UploadHeaders struct {
				Entries []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"entries"`
			} `json:"uploadHeaders"`
		} `graphql:"terraformProviderVersionRegisterPlatformV2(version: $version, input: $input)"`
	}

	archiveChecksum, err := artifact.Checksum(dir)
	if err != nil {
		return errors.Wrap(err, "could not calculate checksum of artifact")
	}

	fmt.Printf("Uploading the artifact for %s/%s\n", *artifact.OS, *artifact.Arch)

	variables := map[string]any{
		"version": graphql.ID(versionID),
		"input": TerraformProviderVersionPlatformInput{
			Architecture:    *artifact.Arch,
			OS:              *artifact.OS,
			ArchiveChecksum: archiveChecksum,
			BinaryChecksum:  artifact.Extra.Checksum.BinarySHA256(),
		},
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	header := http.Header{}
	for _, entry := range mutation.RegisterTerraformProviderVersionPlatform.UploadHeaders.Entries {
		header.Set(entry.Key, entry.Value)
	}

	if err := artifact.Upload(ctx, dir, mutation.RegisterTerraformProviderVersionPlatform.UploadURL, header); err != nil {
		return errors.Wrapf(err, "could not upload artifact: %s", artifact.Name)
	}

	return nil
}
