package zda

import "github.com/shimmeringbee/da"

type manageDeviceCapabilitiesShim struct {
	deviceCapabilityManager DeviceCapabilityManager
}

func (s *manageDeviceCapabilitiesShim) Add(d Device, c da.Capability) {
	s.deviceCapabilityManager.AddCapability(d.Identifier, c)
}

func (s *manageDeviceCapabilitiesShim) Remove(d Device, c da.Capability) {
	s.deviceCapabilityManager.RemoveCapability(d.Identifier, c)
}
