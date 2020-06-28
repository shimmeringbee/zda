package zda

import (
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestZigbeeGateway_DeviceStore(t *testing.T) {
	t.Run("device store performs basic actions", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		id := zigbee.IEEEAddress(0x0102030405060708)

		_, found := zgw.getDevice(id)
		assert.False(t, found)

		iNode := zgw.addNode(id)
		iDev := zgw.addDevice(id, iNode)
		assert.Equal(t, id, iDev.device.Identifier)
		assert.Equal(t, zgw, iDev.device.Gateway)
		assert.Equal(t, []Capability{EnumerateDeviceFlag, LocalDebugFlag}, iDev.device.Capabilities)

		iDev, found = zgw.getDevice(id)
		assert.True(t, found)
		assert.Equal(t, id, iDev.device.Identifier)

		zgw.removeDevice(id)

		_, found = zgw.getDevice(id)
		assert.False(t, found)
	})
}

func TestIEEEAddressEndpoint_String(t *testing.T) {
	t.Run("Appends Endpoint to the end of IEEE address as an identifier", func(t *testing.T) {
		ieee := zigbee.IEEEAddress(0x0102030405060708)
		endpoint := zigbee.Endpoint(0xAA)

		id := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: uint8(endpoint),
		}

		assert.Equal(t, "0102030405060708-aa", id.String())
	})
}
