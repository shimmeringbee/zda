package zda

import "github.com/shimmeringbee/da"

type ComposeDADeviceShim struct {
	gateway da.Gateway
}

func (s *ComposeDADeviceShim) Compose(zdaDevice Device) da.Device {
	return da.BaseDevice{
		DeviceGateway:      s.gateway,
		DeviceIdentifier:   zdaDevice.Identifier,
		DeviceCapabilities: zdaDevice.Capabilities,
	}
}
