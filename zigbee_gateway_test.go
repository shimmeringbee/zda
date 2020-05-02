package zda

import (
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZigbeeGateway_Contract(t *testing.T) {
	t.Run("can be assigned to a da.Gateway", func(t *testing.T) {
		zgw := &ZigbeeGateway{}
		var i interface{} = zgw
		_, ok := i.(Gateway)
		assert.True(t, ok)
	})
}

func TestZigbeeGateway_New(t *testing.T) {
	t.Run("a new gateway that is configured and started, has a self device which is valid", func(t *testing.T) {
		mockProvider := new(zigbee.MockProvider)
		defer mockProvider.AssertExpectations(t)

		expectedIEEE := zigbee.IEEEAddress(0x0102030405060708)
		expectedNetwork := zigbee.NetworkAddress(0xeeff)

		mockProvider.On("AdapterNode").Return(zigbee.Node{
			IEEEAddress:    expectedIEEE,
			NetworkAddress: expectedNetwork,
		})

		zgw := New(mockProvider)

		zgw.Start()
		defer zgw.Stop()

		expectedDevice := Device{
			Gateway:    zgw,
			Identifier: expectedIEEE,
			Capabilities: []Capability{
				DeviceDiscoveryFlag,
			},
		}

		actualDevice := zgw.Self()

		assert.Equal(t, expectedDevice, actualDevice)
	})
}

func TestZigbeeGateway_Devices(t *testing.T) {
	t.Run("devices returns self", func(t *testing.T) {
		mockProvider := new(zigbee.MockProvider)
		defer mockProvider.AssertExpectations(t)

		expectedIEEE := zigbee.IEEEAddress(0x0102030405060708)
		expectedNetwork := zigbee.NetworkAddress(0xeeff)

		mockProvider.On("AdapterNode").Return(zigbee.Node{
			IEEEAddress:    expectedIEEE,
			NetworkAddress: expectedNetwork,
		})

		zgw := New(mockProvider)

		zgw.Start()
		defer zgw.Stop()

		expectedDevice := Device{
			Gateway:    zgw,
			Identifier: expectedIEEE,
			Capabilities: []Capability{
				DeviceDiscoveryFlag,
			},
		}

		expectedDevices := []Device{expectedDevice}
		actualDevices := zgw.Devices()

		assert.Equal(t, expectedDevices, actualDevices)
	})
}
