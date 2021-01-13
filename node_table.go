package zda

import (
	"context"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"math"
	"sync"
)

type nodeTable interface {
	getNode(address zigbee.IEEEAddress) *internalNode
	createNode(address zigbee.IEEEAddress) (*internalNode, bool)
	removeNode(address zigbee.IEEEAddress) bool

	getDevice(identifier IEEEAddressWithSubIdentifier) *internalDevice
	createDevice(identifier IEEEAddressWithSubIdentifier) (*internalDevice, bool)
	createNextDevice(address zigbee.IEEEAddress) *internalDevice
	removeDevice(identifier IEEEAddressWithSubIdentifier) bool

	getDevices() []*internalDevice
	getNodes() []*internalNode
}

type zdaNodeTable struct {
	nodes     map[zigbee.IEEEAddress]*internalNode
	lock      *sync.RWMutex
	callbacks callbacks.Caller
}

func newNodeTable() *zdaNodeTable {
	return &zdaNodeTable{
		nodes: make(map[zigbee.IEEEAddress]*internalNode),
		lock:  &sync.RWMutex{},
	}
}

func (z *zdaNodeTable) getNode(address zigbee.IEEEAddress) *internalNode {
	z.lock.RLock()
	defer z.lock.RUnlock()

	return z.nodes[address]
}

func (z *zdaNodeTable) createNode(address zigbee.IEEEAddress) (*internalNode, bool) {
	z.lock.Lock()
	defer z.lock.Unlock()

	node, alreadyExists := z.nodes[address]
	if !alreadyExists {
		node = &internalNode{
			ieeeAddress:          address,
			mutex:                &sync.RWMutex{},
			devices:              make(map[uint8]*internalDevice),
			endpoints:            make([]zigbee.Endpoint, 0),
			endpointDescriptions: make(map[zigbee.Endpoint]zigbee.EndpointDescription),
			transactionSequences: make(chan uint8, math.MaxUint8),
		}

		for i := uint8(0); i < math.MaxUint8; i++ {
			node.transactionSequences <- i
		}

		z.nodes[address] = node
	}

	return node, !alreadyExists
}

func (z *zdaNodeTable) removeNode(address zigbee.IEEEAddress) bool {
	z.lock.Lock()
	defer z.lock.Unlock()

	_, found := z.nodes[address]
	if found {
		delete(z.nodes, address)
	}

	return found
}

func (z *zdaNodeTable) getDevice(identifier IEEEAddressWithSubIdentifier) *internalDevice {
	node := z.getNode(identifier.IEEEAddress)
	if node == nil {
		return nil
	}

	return node._getDevice(identifier.SubIdentifier)
}

func (n *internalNode) _getDevice(subidentifier uint8) *internalDevice {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.devices[subidentifier]
}

func (z *zdaNodeTable) createDevice(identifier IEEEAddressWithSubIdentifier) (*internalDevice, bool) {
	node := z.getNode(identifier.IEEEAddress)
	if node == nil {
		return nil, false
	}

	dev, created := node._createDevice(identifier.SubIdentifier)

	if created && z.callbacks != nil {
		z.callbacks.Call(context.Background(), internalDeviceAdded{device: dev})
	}

	return dev, created
}

func (n *internalNode) _createDevice(subidentifier uint8) (*internalDevice, bool) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	device, found := n.devices[subidentifier]
	if !found {
		device = &internalDevice{
			subidentifier: subidentifier,
			node:          n,
			mutex:         &sync.RWMutex{},
			endpoints:     []zigbee.Endpoint{},
			capabilities:  []da.Capability{capabilities.EnumerateDeviceFlag, capabilities.DeviceRemovalFlag},
		}

		n.devices[subidentifier] = device
	}

	return device, !found
}

func (z *zdaNodeTable) createNextDevice(address zigbee.IEEEAddress) *internalDevice {
	node := z.getNode(address)
	if node == nil {
		return nil
	}

	dev := node._createNextDevice()

	if z.callbacks != nil {
		z.callbacks.Call(context.Background(), internalDeviceAdded{device: dev})
	}

	return dev
}

func (n *internalNode) _createNextDevice() *internalDevice {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	subidentifier := n._findNextDeviceId()

	device := &internalDevice{
		subidentifier: subidentifier,
		node:          n,
		mutex:         &sync.RWMutex{},
		endpoints:     []zigbee.Endpoint{},
		capabilities:  []da.Capability{capabilities.EnumerateDeviceFlag},
	}

	n.devices[subidentifier] = device

	return device

}

func (n *internalNode) _findNextDeviceId() uint8 {
	var foundIds []uint8

	for id := range n.devices {
		foundIds = append(foundIds, id)
	}

	subId := uint8(0)

	for ; subId < math.MaxUint8; subId++ {
		if isUint8InSlice(foundIds, subId) {
			continue
		}

		break
	}

	return subId
}

func (z *zdaNodeTable) removeDevice(identifier IEEEAddressWithSubIdentifier) bool {
	node := z.getNode(identifier.IEEEAddress)
	if node == nil {
		return false
	}

	dev := node._getDevice(identifier.SubIdentifier)
	removed := node._removeDevice(identifier.SubIdentifier)

	if removed && dev != nil && z.callbacks != nil {
		z.callbacks.Call(context.Background(), internalDeviceRemoved{device: dev})
	}

	return removed
}

func (n *internalNode) _removeDevice(subidentifier uint8) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	_, found := n.devices[subidentifier]
	if found {
		delete(n.devices, subidentifier)
	}

	return found
}

func (z *zdaNodeTable) getDevices() []*internalDevice {
	nodes := z.getNodes()

	var devices []*internalDevice

	for _, iNode := range nodes {
		iNode.mutex.RLock()

		for _, iDev := range iNode.devices {
			devices = append(devices, iDev)
		}

		iNode.mutex.RUnlock()
	}

	return devices
}

func (z *zdaNodeTable) getNodes() []*internalNode {
	z.lock.RLock()
	defer z.lock.RUnlock()

	var nodes []*internalNode

	for _, iNode := range z.nodes {
		nodes = append(nodes, iNode)
	}

	return nodes
}
