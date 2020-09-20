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
	subidentifier uint8
	node          *internalNode
	mutex         *sync.RWMutex

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

func (d *internalDevice) generateIdentifier() IEEEAddressWithSubIdentifier {
	return IEEEAddressWithSubIdentifier{IEEEAddress: d.node.ieeeAddress, SubIdentifier: d.subidentifier}
}

func (d *internalDevice) toDevice(g Gateway) Device {
	return BaseDevice{
		DeviceGateway:      g,
		DeviceIdentifier:   d.generateIdentifier(),
		DeviceCapabilities: d.capabilities,
	}
}

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
