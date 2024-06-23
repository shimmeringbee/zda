package product_information

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProductInformation(t *testing.T) {
	t.Run("has basic capability functions", func(t *testing.T) {
		pi := Implementation{}

		assert.Equal(t, capabilities.ProductInformationFlag, pi.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.ProductInformationFlag], pi.Name())
		assert.Equal(t, "GenericProductInformation", pi.ImplName())
	})

	t.Run("accepts data on attach and returns via Get", func(t *testing.T) {
		pi := NewProductInformation()
		pi.Init(nil, memory.New())

		attached, err := pi.Enumerate(nil, map[string]any{
			"Name":         "NEXUS-7",
			"Manufacturer": "Tyrell Corporation",
			"Serial":       "N7FAA52318",
		})
		assert.True(t, attached)
		assert.NoError(t, err)

		actualInfo, err := pi.Get(nil)
		assert.NoError(t, err)

		expectedInfo := capabilities.ProductInfo{
			Manufacturer: "Tyrell Corporation",
			Name:         "NEXUS-7",
			Serial:       "N7FAA52318",
			Version:      "",
		}

		assert.Equal(t, expectedInfo, actualInfo)
	})

	t.Run("handles failure of data gracefully on new enumeration", func(t *testing.T) {
		pi := NewProductInformation()
		pi.Init(nil, memory.New())

		attached, err := pi.Enumerate(nil, map[string]any{
			"Name":         "NEXUS-7",
			"Manufacturer": "Tyrell Corporation",
			"Serial":       "N7FAA52318",
		})
		assert.True(t, attached)
		assert.NoError(t, err)

		attached, err = pi.Enumerate(nil, map[string]any{
			"Name": 7,
		})
		assert.True(t, attached)
		assert.Error(t, err)
	})

	t.Run("fails to attach if data is not string", func(t *testing.T) {
		pi := NewProductInformation()
		pi.Init(nil, memory.New())

		attached, err := pi.Enumerate(nil, map[string]any{
			"Name": 7,
		})
		assert.False(t, attached)
		assert.Error(t, err)
	})

	t.Run("Capturing state and reloading should result in same output state", func(t *testing.T) {
		s := memory.New()
		pi1 := NewProductInformation()
		pi1.Init(nil, s)

		attached, err := pi1.Enumerate(nil, map[string]any{
			"Name":         "NEXUS-7",
			"Manufacturer": "Tyrell Corporation",
			"Serial":       "N7FAA52318",
			"Version":      "1.0.0",
		})
		assert.True(t, attached)
		assert.NoError(t, err)

		pi2 := NewProductInformation()
		pi2.Init(nil, s)

		attached, err = pi2.Load(context.TODO())
		assert.True(t, attached)
		assert.NoError(t, err)

		out1, _ := pi1.Get(nil)
		out2, _ := pi2.Get(nil)

		assert.Equal(t, out1, out2)
	})

	t.Run("fails to attach if there is no data", func(t *testing.T) {
		pi := NewProductInformation()
		pi.Init(nil, memory.New())

		attached, err := pi.Enumerate(nil, map[string]any{})
		assert.False(t, attached)
		assert.NoError(t, err)
	})

}
