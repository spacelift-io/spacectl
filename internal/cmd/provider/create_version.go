package provider

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func createVersion() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		providerType := cliCtx.String(flagProviderType.Name)
		dir := cliCtx.String(flagGoReleaserDir.Name)

		versionData, err := BuildGoReleaserVersionData(dir)
		if err != nil {
			return errors.Wrap(err, "invalid release data")
		}

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
				SigningKeyID:     cliCtx.String(flagSigningKeyID.Name),
			},
		}

		if err := authenticated.Client.Mutate(cliCtx.Context, &createMutation, variables); err != nil {
			return err
		}

		if err := checksumsFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL); err != nil {
			return errors.Wrap(err, "could not upload checksums file")
		}

		if err := signatureFile.Upload(cliCtx.Context, dir, createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL); err != nil {
			return errors.Wrap(err, "could not upload signature file")
		}

		versionID := createMutation.CreateTerraformProviderVersion.Version.ID

		for i := range versionData.Artifacts {
			if err := registerVersion(cliCtx.Context, dir, versionID, &versionData.Artifacts[i]); err != nil {
				return err
			}
		}

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

		if err := authenticated.Client.Mutate(cliCtx.Context, &changelogMutation, variables); err != nil {
			return errors.Wrap(err, "could not update changelog")
		}

		return nil
	}
}

func registerVersion(ctx context.Context, dir string, versionID string, artifact *GoReleaserArtifact) error {
	var mutation struct {
		RegisterTerraformProviderVersionPlatform string `graphql:"terraformProviderVersionRegisterPlatform(version: $version, input: $input)"`
	}

	if artifact.Arch == nil {
		return errors.New("artifact has no architecture specified")
	}

	if artifact.OS == nil {
		return errors.New("artifact has no operating system specified")
	}

	archiveChecksum, err := artifact.Checksum(dir)
	if err != nil {
		return errors.Wrap(err, "could not calculate checksum of artifact")
	}

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
