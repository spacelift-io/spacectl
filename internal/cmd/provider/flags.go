package provider

import "github.com/urfave/cli/v2"

var flagKeyGenerate = &cli.BoolFlag{
	Name:  "generate",
	Usage: "Generate a new GPG key in the client",
}

var flagKeyEmail = &cli.StringFlag{
	Name:  "email",
	Usage: "Email address associated with the GPG key, if generating a new key",
}

var flagKeyID = &cli.StringFlag{
	Name:     "key",
	Usage:    "ID of the GPG key to revoke",
	Required: true,
}

var flagKeyImport = &cli.BoolFlag{
	Name:  "import",
	Usage: "Import a GPG key from an armored file",
}

var flagKeyPath = &cli.StringFlag{
	Name: "path",
	Usage: "When generating a new key, the path to export the ASCII-armored private key to. " +
		"When importing a key, the path to import the key from.",
	Value: "gpg_key.asc",
}

var flagKeyName = &cli.StringFlag{
	Name:     "name",
	Usage:    "Name of the GPG key to create. If generating a new key, this is the name associated with the key",
	Required: true,
}

var flagProviderType = &cli.StringFlag{
	Name:     "type",
	Usage:    "[Required] Type of the provider",
	Required: true,
}

var flagProviderVersionProtocols = &cli.StringSliceFlag{
	Name:  "protocols",
	Usage: "Terraform plugin protocols supported by the provider",
	Value: cli.NewStringSlice("5.0"),
}

var flagGoReleaserDir = &cli.StringFlag{
	Name:  "goreleaser-dir",
	Usage: "Directory containing the GoReleaser build artifacts",
	Value: "dist",
}

var flagRequiredVersionID = &cli.StringFlag{
	Name:     "version",
	Usage:    "Version of the provider",
	Required: true,
}

var gpgKeyFingerprint = &cli.StringFlag{
	Name:     "gpg-fingerprint",
	Usage:    "ID (fingerprint) of the GPG key used to sign the provider version",
	EnvVars:  []string{"GPG_FINGERPRINT"},
	Required: true,
}
