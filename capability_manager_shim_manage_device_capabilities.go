package zda

import "github.com/shimmeringbee/da"

type manageDeviceCapabilitiesShim struct {
	deviceCapabilityManager DeviceCapabilityManager
}

func (s *manageDeviceCapabilitiesShim) Add(d Device, c da.Capability) {
	s.deviceCapabilityManager.addCapability(d.Identifier, c)
}

func (s *manageDeviceCapabilitiesShim) Remove(d Device, c da.Capability) {
	s.deviceCapabilityManager.removeCapability(d.Identifier, c)
}
