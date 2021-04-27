package zda

import (
	"context"
	"fmt"
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

func (z ZigbeeDeviceRemoval) Remove(ctx context.Context, device da.Device, removalType capabilities.RemovalType) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !device.HasCapability(capabilities.DeviceRemovalFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDev := z.nodeTable.getDevice(device.Identifier().(IEEEAddressWithSubIdentifier))
	if iDev == nil {
		return da.DeviceDoesNotBelongToGatewayError
	}

	switch removalType {
	case capabilities.Request:
		z.logger.LogInfo(ctx, "Requesting removal of device from zigbee provider.", logwrap.Datum("IEEEAddress", iDev.node.ieeeAddress.String()))
		return z.nodeRemover.RequestNodeLeave(ctx, iDev.node.ieeeAddress)
	case capabilities.Force:
		z.logger.LogInfo(ctx, "Requesting forced removal of device from zigbee provider.", logwrap.Datum("IEEEAddress", iDev.node.ieeeAddress.String()))
		return z.nodeRemover.ForceNodeLeave(ctx, iDev.node.ieeeAddress)
	default:
		z.logger.LogError(ctx, "Request removal called with unknown removal type.", logwrap.Datum("IEEEAddress", iDev.node.ieeeAddress.String()), logwrap.Datum("removalType", removalType))
		return fmt.Errorf("remove device called with unknown removal type: %v", removalType)
	}
}
