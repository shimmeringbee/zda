package zda

import (
	. "github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestInternalDevice_toDevice(t *testing.T) {
	t.Run("returns a device with expected parameters", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		id := zigbee.IEEEAddress(0x0102030405060708)

		zgw.nodeTable.createNode(id)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: id, SubIdentifier: 0x01}

		iDev, _ := zgw.nodeTable.createDevice(subId)
		iDev.capabilities = []Capability{Capability(0x01)}

		device := iDev.toDevice(zgw)

		assert.Equal(t, subId, device.Identifier())
		assert.Equal(t, zgw, device.Gateway())
		assert.Equal(t, iDev.capabilities, device.Capabilities())
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
