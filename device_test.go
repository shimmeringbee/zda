package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/generic"
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
			capabilities: map[da.Capability]implcaps.ZDACapability{c: nil},
			m:            &sync.RWMutex{},
		}

		assert.Contains(t, d.Capabilities(), c)
	})

	t.Run("Capability returns the stored capability", func(t *testing.T) {
		c := &generic.ProductInformation{}

		d := device{
			capabilities: map[da.Capability]implcaps.ZDACapability{capabilities.ProductInformationFlag: c},
			m:            &sync.RWMutex{},
		}

		assert.Equal(t, c, d.Capability(capabilities.ProductInformationFlag))
	})
}

func Test_gateway_transmissionLookup(t *testing.T) {
	t.Run("returns details required to transmit to zigbee", func(t *testing.T) {
		expectedAddress := zigbee.GenerateLocalAdministeredIEEEAddress()

		ch := make(chan uint8, 1)
		ch <- 1

		d := &device{
			address: IEEEAddressWithSubIdentifier{IEEEAddress: expectedAddress, SubIdentifier: 1},
			n: &node{
				sequence:  ch,
				useAPSAck: true,
			},
		}

		g := &ZDA{}

		ieee, endpoint, aps, seq := g.transmissionLookup(d, zigbee.ProfileHomeAutomation)

		assert.Equal(t, expectedAddress, ieee)
		assert.Equal(t, zigbee.Endpoint(1), endpoint)
		assert.True(t, aps)
		assert.Equal(t, uint8(1), seq)
	})
}
