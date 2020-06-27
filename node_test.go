package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestZigbeeGateway_NodeStore(t *testing.T) {
	t.Run("device store performs basic actions", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		id := zigbee.IEEEAddress(0x0102030405060708)

		_, found := zgw.getNode(id)
		assert.False(t, found)

		iNode := zgw.addNode(id)
		assert.Equal(t, id, iNode.ieeeAddress)

		iNode, found = zgw.getNode(id)
		assert.True(t, found)
		assert.Equal(t, id, iNode.ieeeAddress)

		zgw.removeNode(id)

		_, found = zgw.getNode(id)
		assert.False(t, found)
	})
}

func TestZigbeeNode_DeviceStore(t *testing.T) {
	t.Run("device store performs basic actions", func(t *testing.T) {
		node := &internalNode{
			mutex:   &sync.RWMutex{},
			devices: map[da.Identifier]*internalDevice{},
		}

		expectedIEEE := zigbee.IEEEAddress(0x01)
		device := &internalDevice{
			device: da.Device{
				Identifier: expectedIEEE,
			},
		}

		_, found := node.getDevice(expectedIEEE)
		assert.False(t, found)

		node.addDevice(device)

		_, found = node.getDevice(expectedIEEE)
		assert.True(t, found)

		devices := node.getDevices()
		assert.Equal(t, 1, len(devices))

		node.removeDevice(device)

		_, found = node.getDevice(expectedIEEE)
		assert.False(t, found)
	})
}
