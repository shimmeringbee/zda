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
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{},
		}

		expectedSubId := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.IEEEAddress(0x01), SubIdentifier: 0x01}
		device := &internalDevice{
			device: da.Device{
				Identifier: expectedSubId,
			},
		}

		_, found := node.getDevice(expectedSubId)
		assert.False(t, found)

		node.addDevice(device)

		_, found = node.getDevice(expectedSubId)
		assert.True(t, found)

		devices := node.getDevices()
		assert.Equal(t, 1, len(devices))

		node.removeDevice(device)

		_, found = node.getDevice(expectedSubId)
		assert.False(t, found)
	})
}

func Test_internalNode_findNextDeviceIdentifier(t *testing.T) {
	t.Run("finds finds next identifier with no devices", func(t *testing.T) {
		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		iNode := internalNode{
			ieeeAddress: ieeeAddress,
			mutex:       &sync.RWMutex{},
			devices:     map[IEEEAddressWithSubIdentifier]*internalDevice{},
		}

		expectedId := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x00}
		actualId := iNode.findNextDeviceIdentifier()

		assert.Equal(t, expectedId, actualId)
	})

	t.Run("finds finds identifier with sub of 1 if 0 is present", func(t *testing.T) {
		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		iNode := internalNode{
			ieeeAddress: ieeeAddress,
			mutex:       &sync.RWMutex{},
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{
				IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0}: nil,
			},
		}

		expectedId := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x01}
		actualId := iNode.findNextDeviceIdentifier()

		assert.Equal(t, expectedId, actualId)
	})

	t.Run("finds finds identifier with sub of 2 if 0, 1, 3 is present", func(t *testing.T) {
		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		iNode := internalNode{
			ieeeAddress: ieeeAddress,
			mutex:       &sync.RWMutex{},
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{
				IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0}: nil,
				IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 1}: nil,
				IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 3}: nil,
			},
		}

		expectedId := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x02}
		actualId := iNode.findNextDeviceIdentifier()

		assert.Equal(t, expectedId, actualId)
	})
}

func Test_internalNode_nextTransactionSequence(t *testing.T) {
	t.Run("receives the next transaction sequence", func(t *testing.T) {
		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		iNode := internalNode{
			ieeeAddress:          ieeeAddress,
			mutex:                &sync.RWMutex{},
			devices:              map[IEEEAddressWithSubIdentifier]*internalDevice{},
			transactionSequences: make(chan uint8, 3),
		}

		iNode.transactionSequences <- 1
		iNode.transactionSequences <- 2
		iNode.transactionSequences <- 3

		assert.Equal(t, uint8(1), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(2), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(3), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(1), iNode.nextTransactionSequence())
	})
}
