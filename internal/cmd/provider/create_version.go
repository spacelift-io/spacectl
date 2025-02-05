package provider

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func createVersion() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		// Assuming that spacectl is ran from the root of the repository,
		// containing the release artifacts in the "dist" directory.
		dir := cliCtx.String(flagGoReleaserDir.Name)

		providerType := cliCtx.String(flagProviderType.Name)
		var useRegisterPlatformV2 bool
		if types, err := mutationTypes(cliCtx.Context); err == nil {
			useRegisterPlatformV2 = types.hasTerraformProviderVersionRegisterPlatformV2Mutation()
		} else {
			fmt.Println("Failed to check for presence of terraformProviderVersionRegisterPlatformV2Mutation ", err.Error())
		}

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

		var createMutation struct {
			CreateTerraformProviderVersion struct {
				SHA256SumsUploadURL    string `graphql:"sha256SumsUploadURL"`
				SHA256SumsSigUploadURL string `graphql:"sha256SumsSigUploadURL"`
				Version                struct {
					ID string `graphql:"id"`
				} `graphql:"version"`
			} `graphql:"terraformProviderVersionCreate(provider: $provider, input: $input)"`
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

		if err := authenticated.Client.Mutate(cliCtx.Context, &createMutation, variables); err != nil {
			return err
		}

		fmt.Println("Uploading the checksums file")
		if err := checksumsFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL, checksumsFile.AWSMetadataHeaders()); err != nil {
			return errors.Wrap(err, "could not upload checksums file")
		}

		fmt.Println("Uploading the signatures file")
		if err := signatureFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL, signatureFile.AWSMetadataHeaders()); err != nil {
			return errors.Wrap(err, "could not upload signature file")
		}

		versionID := createMutation.CreateTerraformProviderVersion.Version.ID

		archives := versionData.Artifacts.Archives()
		for i := range archives {
			if err := archives[i].ValidateFilename(providerType, versionData.Metadata.Version); err != nil {
				return errors.Wrapf(err, "invalid artifact filename: %s", archives[i].Name)
			}

			if useRegisterPlatformV2 {
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
			"description": *versionData.Changelog,
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

type mutationTypesQuery struct {
	Schema struct {
		MutationType struct {
			Fields []mutationTypeField
		}
	} `graphql:"__schema"`
}

type mutationTypeField struct {
	Name string
}

func (q mutationTypesQuery) hasTerraformProviderVersionRegisterPlatformV2Mutation() bool {
	return slices.ContainsFunc(q.Schema.MutationType.Fields, func(field mutationTypeField) bool {
		return field.Name == "terraformProviderVersionRegisterPlatformV2"
	})
}

func mutationTypes(ctx context.Context) (mutationTypesQuery, error) {
	query := mutationTypesQuery{}
	err := authenticated.Client.Query(ctx, &query, nil)
	if err != nil {
		return mutationTypesQuery{}, err
	}
	return query, nil
}

func registerPlatformV2(ctx context.Context, dir string, versionID string, artifact *internal.GoReleaserArtifact) error {
	var mutation struct {
		RegisterTerraformProviderVersionPlatform struct {
			UploadUrl     string `json:"uploadUrl"`
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

	if err := artifact.Upload(ctx, dir, mutation.RegisterTerraformProviderVersionPlatform.UploadUrl, header); err != nil {
		return errors.Wrapf(err, "could not upload artifact: %s", artifact.Name)
	}

	return nil
}
