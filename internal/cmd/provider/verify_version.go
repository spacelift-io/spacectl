package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// alreadyVerifiedMessage is the substring returned by the backend when a
// version has already been verified successfully.
const alreadyVerifiedMessage = "this provider version has already been verified"

func verifyVersion() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		quiet := cliCmd.Bool(flagQuiet.Name)
		versionID := cliCmd.String(flagRequiredVersionID.Name)
		log := func(s string) {
			if !quiet {
				fmt.Print(s)
			}
		}

		var verifyMutation struct {
			Version struct {
				ID                  string  `graphql:"id"`
				VerificationFailure *string `graphql:"verificationFailure"`
			} `graphql:"terraformProviderVersionVerify(version: $version)"`
		}

		log("Verifying the signature and checksums: ")

		if err := authenticated.Client().Mutate(ctx, &verifyMutation, map[string]any{
			"version": graphql.ID(versionID),
		}); err != nil && strings.Contains(err.Error(), alreadyVerifiedMessage) {
			log("already verified, skipping\n")
		} else if err != nil {
			log("failed\n")
			return err
		} else if vf := verifyMutation.Version.VerificationFailure; vf != nil && *vf != "" {
			// A failed verification is returned as data, not as an error, so we
			// surface it as a non-zero exit explicitly.
			log("failed\n")
			return fmt.Errorf("provider version verification failed: %s", *vf)
		} else {
			log("OK\n")
		}

		return nil
	}
}
