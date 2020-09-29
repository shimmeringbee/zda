package zda

import (
	"fmt"
	. "github.com/shimmeringbee/da"
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
}

func (z *ZigbeeGateway) AddCapabilityToDevice(id IEEEAddressWithSubIdentifier, capability Capability) {
	if iDev := z.nodeTable.getDevice(id); iDev != nil {
		iDev.mutex.Lock()
		if !isCapabilityInSlice(iDev.capabilities, capability) {
			iDev.capabilities = append(iDev.capabilities, capability)
		}
		iDev.mutex.Unlock()
	}
}

func (z *ZigbeeGateway) RemoveCapabilityFromDevice(id IEEEAddressWithSubIdentifier, capability Capability) {
	if iDev := z.nodeTable.getDevice(id); iDev != nil {
		iDev.mutex.Lock()

		var newCapabilities []Capability

		for _, existingCapability := range iDev.capabilities {
			if existingCapability != capability {
				newCapabilities = append(newCapabilities, existingCapability)
			}
		}

		iDev.capabilities = newCapabilities

		iDev.mutex.Unlock()
	}
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
