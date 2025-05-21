package provider

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func revokeGPGKey() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		keyID := cliCmd.String(flagKeyID.Name)

		var mutation struct {
			RevokeGPGKey internal.GPGKey `graphql:"gpgKeyRevoke(id: $id)"`
		}

		variables := map[string]any{"id": keyID}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Printf("Revoked GPG key with ID %s", mutation.RevokeGPGKey.ID)

		return nil
	}
}
