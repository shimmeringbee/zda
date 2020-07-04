package zda

import (
	. "github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"math"
	"sync"
)

type internalNode struct {
	ieeeAddress zigbee.IEEEAddress
	mutex       *sync.RWMutex

	devices map[IEEEAddressWithSubIdentifier]*internalDevice

	nodeDesc             zigbee.NodeDescription
	endpoints            []zigbee.Endpoint
	endpointDescriptions map[zigbee.Endpoint]zigbee.EndpointDescription

	transactionSequences chan uint8
	supportsAPSAck       bool
}

func (z *ZigbeeGateway) getNode(ieeeAddress zigbee.IEEEAddress) (*internalNode, bool) {
	z.nodesLock.RLock()
	defer z.nodesLock.RUnlock()

	node, found := z.nodes[ieeeAddress]
	return node, found
}

func (z *ZigbeeGateway) addNode(ieeeAddress zigbee.IEEEAddress) *internalNode {
	z.nodesLock.Lock()
	defer z.nodesLock.Unlock()

	z.nodes[ieeeAddress] = &internalNode{
		ieeeAddress: ieeeAddress,
		mutex:       &sync.RWMutex{},
		devices:     map[IEEEAddressWithSubIdentifier]*internalDevice{},

		endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},

		transactionSequences: make(chan uint8, math.MaxUint8),
		supportsAPSAck:       false,
	}

	for i := uint8(0); i < math.MaxUint8; i++ {
		z.nodes[ieeeAddress].transactionSequences <- i
	}

	return z.nodes[ieeeAddress]
}

func (z *ZigbeeGateway) removeNode(ieeeAddress zigbee.IEEEAddress) {
	z.nodesLock.Lock()
	defer z.nodesLock.Unlock()

	delete(z.nodes, ieeeAddress)
}

func (n *internalNode) findNextDeviceIdentifier() IEEEAddressWithSubIdentifier {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	var foundIds []uint8

	for id, _ := range n.devices {
		foundIds = append(foundIds, id.SubIdentifier)
	}

	subId := uint8(0)

	for ; subId < math.MaxUint8; subId++ {
		if isValueInSlice(foundIds, subId) {
			continue
		}

		break
	}

	return IEEEAddressWithSubIdentifier{IEEEAddress: n.ieeeAddress, SubIdentifier: subId}
}

func isValueInSlice(haystack []uint8, needle uint8) bool {
	for _, piece := range haystack {
		if piece == needle {
			return true
		}
	}

	return false
}

func (n *internalNode) addDevice(zigbeeDevice *internalDevice) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	subId := zigbeeDevice.device.Identifier.(IEEEAddressWithSubIdentifier)
	n.devices[subId] = zigbeeDevice
}

func (n *internalNode) removeDevice(zigbeeDevice *internalDevice) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	subId := zigbeeDevice.device.Identifier.(IEEEAddressWithSubIdentifier)
	delete(n.devices, subId)
}

func (n *internalNode) getDevice(identifier Identifier) (*internalDevice, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	subId := identifier.(IEEEAddressWithSubIdentifier)
	device, found := n.devices[subId]
	return device, found
}

func (n *internalNode) getDevices() []*internalDevice {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	var devices []*internalDevice

	for _, device := range n.devices {
		devices = append(devices, device)
	}

	return devices
}

func (n *internalNode) nextTransactionSequence() uint8 {
	nextSeq := <-n.transactionSequences
	n.transactionSequences <- nextSeq

	return nextSeq
}
