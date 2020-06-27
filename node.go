package zda

import (
	. "github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type internalNode struct {
	ieeeAddress zigbee.IEEEAddress
	mutex       *sync.RWMutex

	devices map[Identifier]*internalDevice

	nodeDesc             zigbee.NodeDescription
	endpoints            []zigbee.Endpoint
	endpointDescriptions map[zigbee.Endpoint]zigbee.EndpointDescription
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
		devices:     map[Identifier]*internalDevice{},

		endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
	}

	return z.nodes[ieeeAddress]
}

func (z *ZigbeeGateway) removeNode(ieeeAddress zigbee.IEEEAddress) {
	z.nodesLock.Lock()
	defer z.nodesLock.Unlock()

	delete(z.nodes, ieeeAddress)
}

func (n *internalNode) addDevice(zigbeeDevice *internalDevice) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.devices[zigbeeDevice.device.Identifier] = zigbeeDevice
}

func (n *internalNode) removeDevice(zigbeeDevice *internalDevice) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	delete(n.devices, zigbeeDevice.device.Identifier)
}

func (n *internalNode) getDevice(identifier Identifier) (*internalDevice, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	device, found := n.devices[identifier]
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
