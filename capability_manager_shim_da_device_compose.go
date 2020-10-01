package zda

import "github.com/shimmeringbee/da"

type composeDADeviceShim struct {
	gateway da.Gateway
}

func (s *composeDADeviceShim) Compose(zdaDevice Device) da.Device {
	return da.BaseDevice{
		DeviceGateway:      s.gateway,
		DeviceIdentifier:   zdaDevice.Identifier,
		DeviceCapabilities: zdaDevice.Capabilities,
	}
}
