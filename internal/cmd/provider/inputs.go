package provider

// TerraformProviderVersionInput represents the input for creating a new
// provider version.
type TerraformProviderVersionInput struct {
	Number           string   `json:"number"`
	ProtocolVersions []string `json:"protocolVersions"`
	SHASumsFileSHA   string   `json:"shaSumsFileSHA"`
	SignatureFileSHA string   `json:"signatureFileSHA"`
	SigningKeyID     string   `json:"signingKeyId"`
}

// TerraformProviderVersionPlatformInput represents the input for registering
// a new platform for a provider version.
type TerraformProviderVersionPlatformInput struct {
	Architecture    string `json:"architecture"`
	ArchiveChecksum string `json:"archiveChecksum"`
	BinaryChecksum  string `json:"binaryChecksum"`
	OS              string `json:"os"`
}
