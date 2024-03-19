package zda

func contains[T comparable](haystack []T, needle T) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}
