package zda

import "github.com/shimmeringbee/da"

type deviceLookupShim struct {
	gateway   da.Gateway
	nodeTable nodeTable
}

func (s *deviceLookupShim) ByDA(d da.Device) (Device, bool) {
	if s.gateway != d.Gateway() {
		return Device{}, false
	}

	addr, ok := d.Identifier().(IEEEAddressWithSubIdentifier)
	if !ok {
		return Device{}, false
	}

	iDev := s.nodeTable.getDevice(addr)
	if iDev == nil {
		return Device{}, false
	}

	return internalDeviceToZDADevice(iDev), true
}

func (s *deviceLookupShim) Self() Device {
	selfDevice := s.gateway.Self()
	return Device{Identifier: selfDevice.Identifier().(IEEEAddressWithSubIdentifier), Capabilities: selfDevice.Capabilities()}
}
