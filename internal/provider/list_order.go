package provider

// orderByPriorIDs orders items like prior, matched by ID, and appends items that are not in prior.
// It keeps list attributes stable when the API returns elements in a different order.
func orderByPriorIDs[T any](items []T, prior []T, id func(T) string) []T {
	if len(prior) == 0 {
		return items
	}
	byID := make(map[string]T, len(items))
	for _, item := range items {
		byID[id(item)] = item
	}
	ordered := make([]T, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, p := range prior {
		key := id(p)
		if item, ok := byID[key]; ok && !seen[key] {
			ordered = append(ordered, item)
			seen[key] = true
		}
	}
	for _, item := range items {
		if key := id(item); !seen[key] {
			ordered = append(ordered, item)
			seen[key] = true
		}
	}
	return ordered
}
