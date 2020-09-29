package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDevice_HasCapability(t *testing.T) {
	t.Run("returns true if device has capability", func(t *testing.T) {
		d := Device{
			Capabilities: []da.Capability{capabilities.EnumerateDeviceFlag},
		}

		assert.True(t, d.HasCapability(capabilities.EnumerateDeviceFlag))
	})

	t.Run("returns false if device does not have capability", func(t *testing.T) {
		d := Device{}

		assert.False(t, d.HasCapability(capabilities.EnumerateDeviceFlag))
	})
}
