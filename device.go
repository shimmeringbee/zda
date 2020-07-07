package zda

import (
	"fmt"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type internalDevice struct {
	// Immutable, no locking required.
	device Device
	node   *internalNode
	mutex  *sync.RWMutex

	// Mutable, locking must be obtained first.
	deviceID      uint16
	deviceVersion uint8
	endpoints     []zigbee.Endpoint

	productInformation ProductInformation
	onOffState         ZigbeeOnOffState
}

func (z *ZigbeeGateway) getDevice(identifier Identifier) (*internalDevice, bool) {
	z.devicesLock.RLock()
	defer z.devicesLock.RUnlock()

	device, found := z.devices[identifier]
	return device, found
}

func (z *ZigbeeGateway) addDevice(identifier Identifier, node *internalNode) *internalDevice {
	device := Device{
		Gateway:      z,
		Identifier:   identifier,
		Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
	}

	iDev := &internalDevice{
		node:   node,
		device: device,
		mutex:  &sync.RWMutex{},
	}

	node.addDevice(iDev)

	z.devicesLock.Lock()
	defer z.devicesLock.Unlock()

	z.devices[identifier] = iDev

	z.sendEvent(DeviceAdded{Device: device})

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

	z.sendEvent(DeviceRemoved{Device: iDevice.device})
}

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
