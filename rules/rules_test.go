package rules

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRule_PopulateParentage(t *testing.T) {
	t.Run("recursively sets the parentage of Rules", func(t *testing.T) {
		r := &Rule{
			Children: []*Rule{
				{
					Children: []*Rule{
						{},
					},
				},
			},
		}

		r.PopulateParentage()

		assert.Equal(t, r, r.Children[0].parent)
		assert.Equal(t, r.Children[0], r.Children[0].Children[0].parent)
	})
}

func TestRule_Match(t *testing.T) {
	t.Run("if no child filters match, then self is returned", func(t *testing.T) {
		notWantedManu := "not-manu"

		r := &Rule{
			Children: []*Rule{
				{
					Filter: Filter{
						ManufacturerName: &notWantedManu,
					},
				},
			},
		}

		foundRule := r.Match(MatchData{})
		assert.Equal(t, r, foundRule)
	})

	t.Run("if child filters match, return child", func(t *testing.T) {
		wantedManu := "manu"

		r := &Rule{
			Children: []*Rule{
				{
					Filter: Filter{
						ManufacturerName: &wantedManu,
					},
				},
			},
		}

		foundRule := r.Match(MatchData{ManufacturerName: wantedManu})
		assert.Equal(t, r.Children[0], foundRule)
	})
}

func TestRule_StringSetting(t *testing.T) {
	t.Run("returns string setting at correct level", func(t *testing.T) {
		r := &Rule{
			Children: []*Rule{
				{
					Settings: map[string]Settings{
						"child": {
							"key": "child",
						},
					},
				},
			},
			Settings: map[string]Settings{
				"root": {
					"key": "root",
				},
			},
		}

		r.PopulateParentage()

		child := r.Children[0]

		assert.Equal(t, "child", child.StringSetting("child", "key", "defValue"))
		assert.Equal(t, "root", child.StringSetting("root", "key", "defValue"))
		assert.Equal(t, "defValue", child.StringSetting("none", "key", "defValue"))
	})

	t.Run("returns int setting at correct level", func(t *testing.T) {
		r := &Rule{
			Children: []*Rule{
				{
					Settings: map[string]Settings{
						"child": {
							"key": 2,
						},
					},
				},
			},
			Settings: map[string]Settings{
				"root": {
					"key": 1,
				},
			},
		}

		r.PopulateParentage()

		child := r.Children[0]

		assert.Equal(t, 2, child.IntSetting("child", "key", 0))
		assert.Equal(t, 1, child.IntSetting("root", "key", 0))
		assert.Equal(t, 0, child.IntSetting("none", "key", 0))
	})

	t.Run("returns float setting at correct level", func(t *testing.T) {
		r := &Rule{
			Children: []*Rule{
				{
					Settings: map[string]Settings{
						"child": {
							"key": 2.0,
						},
					},
				},
			},
			Settings: map[string]Settings{
				"root": {
					"key": 1.0,
				},
			},
		}

		r.PopulateParentage()

		child := r.Children[0]

		assert.Equal(t, 2.0, child.FloatSetting("child", "key", 0.0))
		assert.Equal(t, 1.0, child.FloatSetting("root", "key", 0.0))
		assert.Equal(t, 0.0, child.FloatSetting("none", "key", 0.0))
	})

	t.Run("returns boolean setting at correct level", func(t *testing.T) {
		r := &Rule{
			Children: []*Rule{
				{
					Settings: map[string]Settings{
						"child": {
							"key": true,
						},
					},
				},
			},
			Settings: map[string]Settings{
				"root": {
					"key": false,
				},
			},
		}

		r.PopulateParentage()

		child := r.Children[0]

		assert.Equal(t, true, child.BooleanSetting("child", "key", false))
		assert.Equal(t, false, child.BooleanSetting("root", "key", false))
		assert.Equal(t, false, child.BooleanSetting("none", "key", false))
	})
}

func TestRule_DurationSetting(t *testing.T) {
	t.Run("wraps IntSetting with casting", func(t *testing.T) {
		r := &Rule{
			Settings: map[string]Settings{
				"root": {
					"key": 1000,
				},
			},
		}

		expectedDuration := 1 * time.Second
		actualDuration := r.DurationSetting("root", "key", 100*time.Millisecond)

		assert.Equal(t, expectedDuration, actualDuration)
	})

	t.Run("wraps IntSetting with casting, respecting default duration", func(t *testing.T) {
		r := &Rule{
			Settings: map[string]Settings{},
		}

		expectedDuration := 100 * time.Millisecond
		actualDuration := r.DurationSetting("root", "key", 100*time.Millisecond)

		assert.Equal(t, expectedDuration, actualDuration)
	})
}
