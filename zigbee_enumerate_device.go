package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

type ZigbeeEnumerateDevice struct {
	gateway *ZigbeeGateway
}

func (z *ZigbeeEnumerateDevice) Enumerate(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.EnumerateDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	return nil
}

func (z *ZigbeeEnumerateDevice) Stop() {

}
