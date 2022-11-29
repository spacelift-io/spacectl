package provider

import "github.com/urfave/cli/v2"

var flagProviderType = &cli.StringFlag{
	Name:     "type",
	Usage:    "[Required] Type of the provider",
	Required: true,
}

var flagProviderVersionProtocols = &cli.StringSliceFlag{
	Name:  "protocols",
	Usage: "Protocols supported by the provider",
}

var flagGoReleaserDir = &cli.StringFlag{
	Name:  "goreleaser-dir",
	Usage: "Directory containing the GoReleaser build artifacts",
}

var flagSigningKeyID = &cli.StringFlag{
	Name:     "signing-key-id",
	Usage:    "ID of the signing key used to sign the provider version",
	EnvVars:  []string{"GPG_FINGERPRINT"},
	Required: true,
}
