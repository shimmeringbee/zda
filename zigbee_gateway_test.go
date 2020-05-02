package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZigbeeGateway_Contract(t *testing.T) {
	t.Run("can be assigned to a da.Gateway", func(t *testing.T) {
		zgw := &ZigbeeGateway{}
		var i interface{} = zgw
		_, ok := i.(da.Gateway)
		assert.True(t, ok)
	})
}
