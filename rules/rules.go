package rules

type Rule struct {
	parent      *Rule
	Description string
	Filter      Filter
	Children    []*Rule
	Settings    map[string]Settings
}

func (r *Rule) PopulateParentage() {
	for _, c := range r.Children {
		c.parent = r
		c.PopulateParentage()
	}
}

func (r *Rule) Match(m MatchData) *Rule {
	if !r.Filter.matches(m) {
		return nil
	}

	for _, c := range r.Children {
		if mr := c.Match(m); mr != nil {
			return mr
		}
	}

	return r
}

func (r *Rule) StringSetting(ns string, key string, def string) string {
	s, nsOk := r.Settings[ns]

	if nsOk {
		v, valOk := s.String(key)

		if valOk {
			return v
		}
	}

	if r.parent != nil {
		return r.parent.StringSetting(ns, key, def)
	}

	return def
}

func (r *Rule) IntSetting(ns string, key string, def int) int {
	s, nsOk := r.Settings[ns]

	if nsOk {
		v, valOk := s.Int(key)

		if valOk {
			return v
		}
	}

	if r.parent != nil {
		return r.parent.IntSetting(ns, key, def)
	}

	return def
}

func (r *Rule) FloatSetting(ns string, key string, def float64) float64 {
	s, nsOk := r.Settings[ns]

	if nsOk {
		v, valOk := s.Float(key)

		if valOk {
			return v
		}
	}

	if r.parent != nil {
		return r.parent.FloatSetting(ns, key, def)
	}

	return def
}

func (r *Rule) BooleanSetting(ns string, key string, def bool) bool {
	s, nsOk := r.Settings[ns]

	if nsOk {
		v, valOk := s.Boolean(key)

		if valOk {
			return v
		}
	}

	if r.parent != nil {
		return r.parent.BooleanSetting(ns, key, def)
	}

	return def
}
