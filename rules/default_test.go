package rules

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefault(t *testing.T) {
	t.Run("default rules can be loaded and pass compilation", func(t *testing.T) {
		e := New()

		err := e.LoadFS(Embedded)
		assert.NoError(t, err)

		err = e.CompileRules()
		assert.NoError(t, err)
	})
}
