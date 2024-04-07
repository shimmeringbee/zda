package temperature_sensor

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImplementation_BaseFunctions(t *testing.T) {
	t.Run("basic static functions respond correctly", func(t *testing.T) {
		i := NewTemperatureSensor()

		assert.Equal(t, capabilities.TemperatureSensorFlag, i.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.TemperatureSensorFlag], i.Name())
		assert.Equal(t, "ZCLTemperatureSensor", i.ImplName())
	})
}
