package slices

// Filter return items that pass the filter function.
func Filter[T any](src []T, predicate func(item T) bool) []T {
	var dst []T
	for _, item := range src {
		if predicate(item) {
			dst = append(dst, item)
		}
	}

	return dst
}
