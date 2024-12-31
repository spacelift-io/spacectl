package provider

import (
	"fmt"
	"os"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func addGPGKey() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		keyGenerate := cliCtx.Bool(flagKeyGenerate.Name)
		keyImport := cliCtx.Bool(flagKeyImport.Name)

		if keyGenerate == keyImport {
			return fmt.Errorf("either --%s or --%s must be specified", flagKeyGenerate.Name, flagKeyImport.Name)
		}

		keyName := cliCtx.String(flagKeyName.Name)
		keyPath := cliCtx.String(flagKeyPath.Name)

		var asciiArmor string
		var err error

		if keyGenerate {
			asciiArmor, err = generateGPGKey(cliCtx, keyName, keyPath)
		} else {
			asciiArmor, err = importGPGKey(cliCtx, keyPath)
		}

		if err != nil {
			return err
		}

		var mutation struct {
			CreateGPGKey internal.GPGKey `graphql:"gpgKeyCreate(name: $name, asciiArmor: $asciiArmor)"`
		}

		variables := map[string]any{
			"name":       keyName,
			"asciiArmor": asciiArmor,
		}

		if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
			return err
		}

		fmt.Printf("Created GPG key with ID %s", mutation.CreateGPGKey.ID)

		return nil
	}
}

func generateGPGKey(cliCtx *cli.Context, keyName, keyPath string) (string, error) {
	email := cliCtx.String(flagKeyEmail.Name)
	if email == "" {
		return "", fmt.Errorf("--%s must be specified", flagKeyEmail.Name)
	}

	key, err := crypto.GenerateKey(keyName, email, "rsa", 4096)
	if err != nil {
		return "", fmt.Errorf("failed to generate GPG key: %w", err)
	}

	privateArmor, err := key.Armor()
	if err != nil {
		return "", fmt.Errorf("failed to generate private key armor: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(privateArmor), 0600); err != nil {
		return "", fmt.Errorf("failed to write private key to %s: %w", keyPath, err)
	}

	fmt.Println("ASCII-armored private key written to", keyPath)

	return key.GetArmoredPublicKey()
}

func importGPGKey(_ *cli.Context, keyPath string) (string, error) {
	// #nosec G304
	bytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to open GPG key file %s: %w", keyPath, err)
	}

	key, err := crypto.NewKeyFromArmored(string(bytes))
	if err != nil {
		return "", fmt.Errorf("failed to import GPG key: %w", err)
	}

	return key.GetArmoredPublicKey()
}
