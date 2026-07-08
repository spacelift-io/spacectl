package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

const pgpArmorPrefix = "-----BEGIN PGP"

func createVersion(useHeadersFromAPI, verifyVersion bool) cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		// Assuming that spacectl is ran from the root of the repository,
		// containing the release artifacts in the "dist" directory.
		dir := cliCmd.String(flagGoReleaserDir.Name)

		providerType := cliCmd.String(flagProviderType.Name)
		quiet := cliCmd.Bool(flagQuiet.Name)
		signingKeyID := cliCmd.String(flagGPGKeyID.Name)

		log := func(format string, a ...any) {
			if !quiet {
				fmt.Printf(format, a...)
			}
		}

		log("Retrieving release data from %s\n", dir)
		versionData, err := internal.BuildGoReleaserVersionData(dir)
		if err != nil {
			return errors.Wrap(err, "invalid release data")
		}

		log("Creating version %s\n", versionData.Metadata.Version)

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

		extractedKeyID, err := getSignerKeyID(signatureFile.Path)
		if err != nil {
			if signingKeyID == "" {
				return errors.Wrap(err, "could not determine signing key ID, specify it with --gpg-key-id to override the key ID")
			}
			log("Could not determine signing key ID, using the key ID specified with --gpg-key-id: %s\n", signingKeyID)
		} else if signingKeyID != "" && !strings.EqualFold(signingKeyID, extractedKeyID) {
			log("Extracted signing key ID %s does not match the key ID %s specified with --gpg-key-id, using %s\n",
				extractedKeyID, signingKeyID, signingKeyID)
		}

		if signingKeyID == "" {
			signingKeyID = extractedKeyID
			log("Extracted signing key ID: %s\n", signingKeyID)
		}

		signatureFileChecksum, err := signatureFile.Checksum(dir)
		if err != nil {
			return errors.Wrap(err, "could not calculate checksum of signature file")
		}

		variables := map[string]any{
			"provider": graphql.ID(providerType),
			"input": TerraformProviderVersionInput{
				Number:           versionData.Metadata.Version,
				ProtocolVersions: cliCmd.StringSlice(flagProviderVersionProtocols.Name),
				SHASumsFileSHA:   checksumsFileChecksum,
				SignatureFileSHA: signatureFileChecksum,
				SigningKeyID:     signingKeyID,
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

			if err := authenticated.Client().Mutate(ctx, &createMutation, variables); err != nil {
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

			if err := authenticated.Client().Mutate(ctx, &createMutation, variables); err != nil {
				return err
			}

			sha256SumsUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsUploadURL
			sha256SumsUploadHeaders = checksumsFile.AWSMetadataHeaders()

			sha256SumsSigUploadURL = createMutation.CreateTerraformProviderVersion.SHA256SumsSigUploadURL
			sha256SumsSigUploadHeaders = signatureFile.AWSMetadataHeaders()

			versionID = createMutation.CreateTerraformProviderVersion.Version.ID
		}

		log("Uploading the checksums file\n")
		if err := checksumsFile.Upload(ctx, dir, sha256SumsUploadURL, sha256SumsUploadHeaders); err != nil {
			return errors.Wrap(err, "could not upload checksums file")
		}

		log("Uploading the signatures file\n")
		if err := signatureFile.Upload(ctx, dir, sha256SumsSigUploadURL, sha256SumsSigUploadHeaders); err != nil {
			return errors.Wrap(err, "could not upload signature file")
		}

		archives := versionData.Artifacts.Archives()
		for i := range archives {
			if err := archives[i].ValidateFilename(providerType, versionData.Metadata.Version); err != nil {
				return errors.Wrapf(err, "invalid artifact filename: %s", archives[i].Name)
			}

			if useHeadersFromAPI {
				if err := registerPlatformV2(ctx, dir, versionID, &archives[i], log); err != nil {
					return err
				}
			} else {
				if err := registerPlatform(ctx, dir, versionID, &archives[i], log); err != nil {
					return err
				}
			}
		}

		// The terraformProviderVersionVerify mutation was only introduced in newer
		// backend versions, so we skip verification when running against older ones.
		if verifyVersion {
			var verifyMutation struct {
				Version struct {
					ID                  string  `graphql:"id"`
					VerificationFailure *string `graphql:"verificationFailure"`
				} `graphql:"terraformProviderVersionVerify(version: $version)"`
			}

			log("Verifying the signature and checksums: ")

			var verification string
			if err = authenticated.Client().Mutate(ctx, &verifyMutation, map[string]any{
				"version": graphql.ID(versionID),
			}); err != nil {
				verification = "verified: failed"
				log("%s\n", err)
			} else if vf := verifyMutation.Version.VerificationFailure; vf != nil && *vf != "" {
				verification = "verified: failed"
				log("%s\n", *vf)
			} else {
				verification = "verified: valid"
				log("OK\n")
			}

			log("Draft version %s created (%s)\n", versionID, verification)
		} else {
			log("Draft version %s created\n", versionID)
		}

		if versionData.Changelog == nil {
			if quiet {
				fmt.Print(versionID)
			}
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

		log("Uploading the changelog\n")

		if err := authenticated.Client().Mutate(ctx, &changelogMutation, variables); err != nil {
			return errors.Wrap(err, "could not update changelog")
		}

		if quiet {
			fmt.Print(versionID)
		}
		return nil
	}
}

// deprecated, use registerPlatformV2 instead.
func registerPlatform(ctx context.Context, dir string, versionID string, artifact *internal.GoReleaserArtifact, log func(string, ...any)) error {
	var mutation struct {
		RegisterTerraformProviderVersionPlatform string `graphql:"terraformProviderVersionRegisterPlatform(version: $version, input: $input)"`
	}

	archiveChecksum, err := artifact.Checksum(dir)
	if err != nil {
		return errors.Wrap(err, "could not calculate checksum of artifact")
	}

	log("Uploading the artifact for %s/%s\n", *artifact.OS, *artifact.Arch)

	variables := map[string]any{
		"version": graphql.ID(versionID),
		"input": TerraformProviderVersionPlatformInput{
			Architecture:    *artifact.Arch,
			OS:              *artifact.OS,
			ArchiveChecksum: archiveChecksum,
			BinaryChecksum:  artifact.Extra.Checksum.BinarySHA256(),
		},
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	if err := artifact.Upload(ctx, dir, mutation.RegisterTerraformProviderVersionPlatform, artifact.AWSMetadataHeaders()); err != nil {
		return errors.Wrapf(err, "could not upload artifact: %s", artifact.Name)
	}

	return nil
}

func registerPlatformV2(ctx context.Context, dir string, versionID string, artifact *internal.GoReleaserArtifact, log func(string, ...any)) error {
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

	log("Uploading the artifact for %s/%s\n", *artifact.OS, *artifact.Arch)

	variables := map[string]any{
		"version": graphql.ID(versionID),
		"input": TerraformProviderVersionPlatformInput{
			Architecture:    *artifact.Arch,
			OS:              *artifact.OS,
			ArchiveChecksum: archiveChecksum,
			BinaryChecksum:  artifact.Extra.Checksum.BinarySHA256(),
		},
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
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

func getSignerKeyID(sigPath string) (string, error) {
	// #nosec G304
	data, err := os.ReadFile(sigPath)
	if err != nil {
		return "", err
	}

	var signature *crypto.PGPSignature
	if strings.HasPrefix(strings.TrimSpace(string(data)), pgpArmorPrefix) {
		if signature, err = crypto.NewPGPSignatureFromArmored(string(data)); err != nil {
			return "", fmt.Errorf("failed to decode armored signature: %w", err)
		}
	} else {
		signature = crypto.NewPGPSignature(data)
	}

	keyIDs, ok := signature.GetHexSignatureKeyIDs()
	if !ok || len(keyIDs) == 0 {
		return "", fmt.Errorf("no signature packet or key ID found in %s", sigPath)
	}

	// We only support long (16-character) uppercase key IDs.
	return strings.ToUpper(keyIDs[0]), nil
}
