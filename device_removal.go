package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
)

type deviceRemoval struct {
	node        *node
	logger      logwrap.Logger
	nodeRemover zigbee.NodeRemover
}

func (z deviceRemoval) Capability() da.Capability {
	return capabilities.DeviceRemovalFlag
}

func (z deviceRemoval) Name() string {
	return capabilities.StandardNames[z.Capability()]
}

func (z deviceRemoval) Remove(ctx context.Context, removalType capabilities.RemovalType) error {
	switch removalType {
	case capabilities.Request:
		z.logger.LogInfo(ctx, "Requesting removal of device from zigbee provider.", logwrap.Datum("IEEEAddress", z.node.address.String()))
		return z.nodeRemover.RequestNodeLeave(ctx, z.node.address)
	case capabilities.Force:
		z.logger.LogInfo(ctx, "Requesting forced removal of device from zigbee provider.", logwrap.Datum("IEEEAddress", z.node.address.String()))
		return z.nodeRemover.ForceNodeLeave(ctx, z.node.address)
	default:
		z.logger.LogError(ctx, "Request removal called with unknown removal type.", logwrap.Datum("IEEEAddress", z.node.address.String()), logwrap.Datum("removalType", removalType))
		return fmt.Errorf("remove device called with unknown removal type: %v", removalType)
	}
}

var _ capabilities.DeviceRemoval = (*deviceRemoval)(nil)
var _ da.BasicCapability = (*deviceRemoval)(nil)
