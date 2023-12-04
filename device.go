package zda

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type device struct {
	// Immutable data.
	address IEEEAddressWithSubIdentifier
	gw      da.Gateway
	m       *sync.RWMutex

	// Mutable data, obtain lock first.
	deviceId     uint16
	capabilities map[da.Capability]da.BasicCapability
	productData  productData
}

func (d device) Capability(capability da.Capability) da.BasicCapability {
	d.m.RLock()
	defer d.m.RUnlock()

	return d.capabilities[capability]
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

	var capabilities []da.Capability

	for k := range d.capabilities {
		capabilities = append(capabilities, k)
	}

	return capabilities
}

var _ da.Device = (*device)(nil)

type IEEEAddressWithSubIdentifier struct {
	IEEEAddress   zigbee.IEEEAddress
	SubIdentifier uint8
}

func (a IEEEAddressWithSubIdentifier) String() string {
	return fmt.Sprintf("%s-%02x", a.IEEEAddress, a.SubIdentifier)
}
