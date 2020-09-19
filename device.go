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
	identifier Identifier
	node       *internalNode
	mutex      *sync.RWMutex

	// Mutable, locking must be obtained first.
	deviceID      uint16
	deviceVersion uint8
	endpoints     []zigbee.Endpoint

	capabilities []Capability

	productInformation ProductInformation
	onOffState         ZigbeeOnOffState
}

func (d *internalDevice) addCapability(capability Capability) {
	if !isCapabilityInSlice(d.capabilities, capability) {
		d.capabilities = append(d.capabilities, capability)
	}
}

func (d *internalDevice) removeCapability(capability Capability) {
	var newCapabilities []Capability

	for _, existingCapability := range d.capabilities {
		if existingCapability != capability {
			newCapabilities = append(newCapabilities, existingCapability)
		}
	}

	d.capabilities = newCapabilities
}

func (d *internalDevice) toDevice() Device {
	return BaseDevice{
		DeviceGateway:      d.node.gateway,
		DeviceIdentifier:   d.identifier,
		DeviceCapabilities: d.capabilities,
	}
}

func (z *ZigbeeGateway) getDevice(identifier Identifier) (*internalDevice, bool) {
	z.devicesLock.RLock()
	defer z.devicesLock.RUnlock()

	device, found := z.devices[identifier]
	return device, found
}

func (z *ZigbeeGateway) addDevice(identifier Identifier, node *internalNode) *internalDevice {
	iDev := &internalDevice{
		node:         node,
		identifier:   identifier,
		mutex:        &sync.RWMutex{},
		capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
	}

	node.addDevice(iDev)

	z.devicesLock.Lock()
	defer z.devicesLock.Unlock()

	z.devices[identifier] = iDev

	z.sendEvent(DeviceAdded{Device: iDev.toDevice()})

	return z.devices[identifier]
}

func (z *ZigbeeGateway) removeDevice(identifier Identifier) {
	iDev, found := z.getDevice(identifier)

	if found {
		iDev.node.removeDevice(iDev)
	}

	z.devicesLock.Lock()
	defer z.devicesLock.Unlock()

	delete(z.devices, identifier)

	z.sendEvent(DeviceRemoved{Device: iDev.toDevice()})
}

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
