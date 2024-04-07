package implcaps

func Get[T any](m map[string]any, k string, def T) T {
	if v, ok := m[k]; ok {
		if cV, ok := v.(T); ok {
			return cV
		}
	}

	return def
}
