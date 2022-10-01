package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func Test_device(t *testing.T) {
	t.Run("returns the gateway and address the device was configured with", func(t *testing.T) {
		gw := &mocks.Gateway{}

		expectedAddr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 1,
		}

		d := device{
			address: expectedAddr,
			gw:      gw,
		}

		assert.Equal(t, gw, d.gw)
		assert.Equal(t, expectedAddr, d.Identifier())
	})

	t.Run("verifies that the devices capability functions behave as expected", func(t *testing.T) {
		c := da.Capability(0x01)

		d := device{
			capabilities: []da.Capability{c},
			m:            &sync.RWMutex{},
		}

		assert.True(t, d.HasCapability(c))
		assert.Contains(t, d.Capabilities(), c)
	})
}
