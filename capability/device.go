package capability

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
)

type Device struct {
	Identifier   zda.IEEEAddressWithSubIdentifier
	Capabilities []da.Capability
	Endpoints    map[zigbee.Endpoint]zigbee.EndpointDescription
}

func (d Device) HasCapability(c da.Capability) bool {
	for _, pC := range d.Capabilities {
		if pC == c {
			return true
		}
	}

	return false
}
