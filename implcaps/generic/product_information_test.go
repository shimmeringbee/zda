package generic

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestProductInformation(t *testing.T) {
	t.Run("has basic capability functions", func(t *testing.T) {
		pi := ProductInformation{}

		assert.Equal(t, capabilities.ProductInformationFlag, pi.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.ProductInformationFlag], pi.Name())
		assert.Equal(t, "GenericProductInformation", pi.ImplName())
	})

	t.Run("accepts data on attach and returns via Get", func(t *testing.T) {
		pi := ProductInformation{m: &sync.RWMutex{}}

		attached, err := pi.Attach(nil, nil, implcaps.Enumeration, map[string]interface{}{
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
		pi := ProductInformation{m: &sync.RWMutex{}}

		attached, err := pi.Attach(nil, nil, implcaps.Enumeration, map[string]interface{}{
			"Name":         "NEXUS-7",
			"Manufacturer": "Tyrell Corporation",
			"Serial":       "N7FAA52318",
		})
		assert.True(t, attached)
		assert.NoError(t, err)

		attached, err = pi.Attach(nil, nil, implcaps.Enumeration, map[string]interface{}{
			"Name": 7,
		})
		assert.True(t, attached)
		assert.Error(t, err)
	})

	t.Run("fails to attach if data is not string", func(t *testing.T) {
		pi := ProductInformation{m: &sync.RWMutex{}}

		attached, err := pi.Attach(nil, nil, implcaps.Enumeration, map[string]interface{}{
			"Name": 7,
		})
		assert.False(t, attached)
		assert.Error(t, err)
	})

	t.Run("Capturing state and reloading should result in same output state", func(t *testing.T) {
		pi1 := ProductInformation{m: &sync.RWMutex{}}

		attached, err := pi1.Attach(nil, nil, implcaps.Enumeration, map[string]interface{}{
			"Name":         "NEXUS-7",
			"Manufacturer": "Tyrell Corporation",
			"Serial":       "N7FAA52318",
			"Version":      "1.0.0",
		})
		assert.True(t, attached)
		assert.NoError(t, err)

		state := pi1.State()

		pi2 := ProductInformation{m: &sync.RWMutex{}}
		attached, err = pi2.Attach(nil, nil, implcaps.Load, state)
		assert.True(t, attached)
		assert.NoError(t, err)

		out1, _ := pi1.Get(nil)
		out2, _ := pi2.Get(nil)

		assert.Equal(t, out1, out2)
	})
}
