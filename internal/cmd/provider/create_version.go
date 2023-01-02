package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func createVersion() cli.ActionFunc {
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
		if err := checksumsFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL); err != nil {
			return errors.Wrap(err, "could not upload checksums file")
		}

		fmt.Println("Uploading the signatures file")
		if err := signatureFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL); err != nil {
			return errors.Wrap(err, "could not upload signature file")
		}

		versionID := createMutation.CreateTerraformProviderVersion.Version.ID

		archives := versionData.Artifacts.Archives()
		for i := range archives {
			if err := archives[i].ValidateFilename(providerType, versionData.Metadata.Version); err != nil {
				return errors.Wrapf(err, "invalid artifact filename: %s", archives[i].Name)
			}

			if err := registerPlatform(cliCtx.Context, dir, versionID, &archives[i]); err != nil {
				return err
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

	if err := artifact.Upload(ctx, dir, mutation.RegisterTerraformProviderVersionPlatform); err != nil {
		return errors.Wrapf(err, "could not upload artifact: %s", artifact.Name)
	}

	return nil
}
