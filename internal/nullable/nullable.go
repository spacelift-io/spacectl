package nullable

// OfValue returns a pointer to the value.
func OfValue[T any](value T) *T {
	return &value
}

// GetValue returns the value of the pointer, or the zero value if the pointer is nil.
func GetValue[T any](value *T) T {
	var defaultValue T
	if value != nil {
		defaultValue = *value
	}
	return defaultValue
}
