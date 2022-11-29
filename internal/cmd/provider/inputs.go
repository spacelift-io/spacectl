package provider

type TerraformProviderVersionInput struct {
	Number           string   `json:"number"`
	ProtocolVersions []string `json:"protocol_versions"`
	SHASumsFileSHA   string   `json:"shaSumsFileSHA"`
	SignatureFileSHA string   `json:"signatureFileSHA"`
	SigningKeyID     string   `json:"signingKeyId"`
}

type TerraformProviderVersionPlatformInput struct {
	Architecture    string `json:"architecture"`
	ArchiveChecksum string `json:"archiveChecksum"`
	BinaryChecksum  string `json:"binaryChecksum"`
	OS              string `json:"os"`
}
