package blueprint

// BlueprintStackCreateInputPair represents a key-value pair for a blueprint input.
type BlueprintStackCreateInputPair struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// BlueprintStackCreateInput represents the input for creating a new stack from a blueprint.
type BlueprintStackCreateInput struct {
	TemplateInputs []BlueprintStackCreateInputPair `json:"templateInputs"`
}
