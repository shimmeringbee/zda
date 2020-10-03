package rules

type Settings map[string]interface{}

func (s Settings) String(k string) (string, bool) {
	val, found := s[k]

	if found {
		s, ok := val.(string)
		return s, ok
	} else {
		return "", false
	}
}

func (s Settings) Boolean(k string) (bool, bool) {
	val, found := s[k]

	if found {
		b, ok := val.(bool)
		return b, ok
	} else {
		return false, false
	}
}

func (s Settings) Int(k string) (int, bool) {
	val, found := s[k]

	if found {
		i, ok := val.(int)
		return i, ok
	} else {
		return 0, false
	}
}

func (s Settings) Float(k string) (float64, bool) {
	val, found := s[k]

	if found {
		f, ok := val.(float64)
		return f, ok
	} else {
		return 0.0, false
	}
}
