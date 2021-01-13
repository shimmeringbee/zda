package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
)

var _ capabilities.DeviceRemoval = (*ZigbeeDeviceRemoval)(nil)

type ZigbeeDeviceRemoval struct {
	logger logwrap.Logger

	gateway     da.Gateway
	nodeTable   nodeTable
	nodeRemover zigbee.NodeRemover
}

func (z ZigbeeDeviceRemoval) Capability() da.Capability {
	return capabilities.DeviceRemovalFlag
}

func (z ZigbeeDeviceRemoval) Name() string {
	return capabilities.StandardNames[z.Capability()]
}

func (z ZigbeeDeviceRemoval) Remove(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !device.HasCapability(capabilities.DeviceRemovalFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDev := z.nodeTable.getDevice(device.Identifier().(IEEEAddressWithSubIdentifier))
	if iDev == nil {
		return da.DeviceDoesNotBelongToGatewayError
	}

	z.logger.LogInfo(ctx, "Requesting removal of device from zigbee provider.", logwrap.Datum("IEEEAddress", iDev.node.ieeeAddress.String()))
	return z.nodeRemover.RemoveNode(ctx, iDev.node.ieeeAddress)
}
