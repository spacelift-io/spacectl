package provider

import (
	"fmt"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func revokeGPGKey() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		keyID := cliCtx.String(flagKeyID.Name)

		var mutation struct {
			RevokeGPGKey internal.GPGKey `graphql:"gpgKeyRevoke(id: $id)"`
		}

		variables := map[string]any{"id": keyID}

		if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
			return err
		}

		fmt.Printf("Revoked GPG key with ID %s", mutation.RevokeGPGKey.ID)

		return nil
	}
}
