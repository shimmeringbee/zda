package zda

import (
	"fmt"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type internalDevice struct {
	device Device
	node   *internalNode
	mutex  *sync.RWMutex

	deviceID      uint16
	deviceVersion uint8
	endpoints     []zigbee.Endpoint
}

func (z *ZigbeeGateway) getDevice(identifier Identifier) (*internalDevice, bool) {
	z.devicesLock.RLock()
	defer z.devicesLock.RUnlock()

	device, found := z.devices[identifier]
	return device, found
}

func (z *ZigbeeGateway) addDevice(identifier Identifier, node *internalNode) *internalDevice {
	z.devicesLock.Lock()
	defer z.devicesLock.Unlock()

	device := Device{
		Gateway:      z,
		Identifier:   identifier,
		Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
	}

	zigbeeDevice := &internalDevice{
		node:   node,
		device: device,
		mutex:  &sync.RWMutex{},
	}

	node.addDevice(zigbeeDevice)
	z.devices[identifier] = zigbeeDevice

	return z.devices[identifier]
}

func (z *ZigbeeGateway) removeDevice(identifier Identifier) {
	iDevice, found := z.getDevice(identifier)

	if found {
		iDevice.node.removeDevice(iDevice)
	}

	z.devicesLock.Lock()
	defer z.devicesLock.Unlock()

	delete(z.devices, identifier)
}

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
