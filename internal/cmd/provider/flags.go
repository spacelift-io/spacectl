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
	Value: cli.NewStringSlice("5.0"),
}

var flagGoReleaserDir = &cli.StringFlag{
	Name:  "goreleaser-dir",
	Usage: "Directory containing the GoReleaser build artifacts",
	Value: "dist",
}

var flagSigningKeyID = &cli.StringFlag{
	Name:     "siging-key-id",
	Usage:    "ID of the signing key used to sign the provider version",
	EnvVars:  []string{"GPG_FINGERPRINT"},
	Required: true,
}
