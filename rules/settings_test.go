package rules

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSettings(t *testing.T) {
	t.Run("a string setting is only returned successfully by String()", func(t *testing.T) {
		k := "key"
		v := "string"
		s := Settings{k: v}

		val, ok := s.String(k)
		assert.Equal(t, v, val)
		assert.True(t, ok)

		_, ok = s.Int(k)
		assert.False(t, ok)

		_, ok = s.Boolean(k)
		assert.False(t, ok)

		_, ok = s.Float(k)
		assert.False(t, ok)
	})

	t.Run("a int setting is only returned successfully by Int()", func(t *testing.T) {
		k := "key"
		v := 2
		s := Settings{k: v}

		_, ok := s.String(k)
		assert.False(t, ok)

		val, ok := s.Int(k)
		assert.True(t, ok)
		assert.Equal(t, v, val)

		_, ok = s.Boolean(k)
		assert.False(t, ok)

		_, ok = s.Float(k)
		assert.False(t, ok)
	})

	t.Run("a float setting is only returned successfully by Float()", func(t *testing.T) {
		k := "key"
		v := 2.0
		s := Settings{k: v}

		_, ok := s.String(k)
		assert.False(t, ok)

		_, ok = s.Int(k)
		assert.False(t, ok)

		_, ok = s.Boolean(k)
		assert.False(t, ok)

		val, ok := s.Float(k)
		assert.True(t, ok)
		assert.Equal(t, v, val)
	})

	t.Run("a boolean setting is only returned successfully by Boolean()", func(t *testing.T) {
		k := "key"
		v := true
		s := Settings{k: v}

		_, ok := s.String(k)
		assert.False(t, ok)

		_, ok = s.Int(k)
		assert.False(t, ok)

		val, ok := s.Boolean(k)
		assert.True(t, ok)
		assert.Equal(t, v, val)

		_, ok = s.Float(k)
		assert.False(t, ok)
	})
}
