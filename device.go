package zda

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type device struct {
	// Immutable data.
	address IEEEAddressWithSubIdentifier
	gw      da.Gateway
	m       *sync.RWMutex
	eda     *enumeratedDeviceAttachment
	dr      *deviceRemoval

	// Mutable data, obtain lock first.
	deviceId     uint16
	capabilities map[da.Capability]implcaps.ZDACapability
	productData  productData
}

func (d device) Capability(capability da.Capability) da.BasicCapability {
	switch capability {
	case capabilities.EnumerateDeviceFlag:
		return d.eda
	case capabilities.DeviceRemovalFlag:
		return d.dr
	default:
		d.m.RLock()
		defer d.m.RUnlock()
		return d.capabilities[capability]
	}
}

func (d device) Gateway() da.Gateway {
	return d.gw
}

func (d device) Identifier() da.Identifier {
	return d.address
}

func (d device) Capabilities() []da.Capability {
	d.m.RLock()
	defer d.m.RUnlock()

	var caps []da.Capability

	for k := range d.capabilities {
		caps = append(caps, k)
	}

	if d.eda != nil {
		caps = append(caps, capabilities.EnumerateDeviceFlag)
	}

	if d.dr != nil {
		caps = append(caps, capabilities.DeviceRemovalFlag)
	}

	return caps
}

var _ da.Device = (*device)(nil)

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
