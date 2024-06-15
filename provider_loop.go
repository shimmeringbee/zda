package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
)

func (z *ZDA) providerLoop() {
	z.providerLoad()

	for {
		event, err := z.provider.ReadEvent(z.ctx)

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			if errors.Is(err, context.Canceled) {
				z.logger.LogInfo(z.ctx, "Provider loop terminating due to cancelled context.")
			} else {
				z.logger.LogError(z.ctx, "Failed to read event from Zigbee provider.", logwrap.Err(err))
			}
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			z.receiveNodeJoinEvent(e)
		case zigbee.NodeLeaveEvent:
			z.receiveNodeLeaveEvent(e)
		case zigbee.NodeIncomingMessageEvent:
			z.receiveNodeIncomingMessageEvent(e)
		}
	}
}

func (z *ZDA) receiveNodeJoinEvent(e zigbee.NodeJoinEvent) {
	z.logger.LogInfo(z.ctx, "Node has joined zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

	if n, created := z.createNode(e.IEEEAddress); created {
		d := z.createNextDevice(n)
		z.logger.LogInfo(z.ctx, "Created default device.", logwrap.Datum("Identifier", d.address.String()))

		if err := z.callbacks.Call(z.ctx, nodeJoin{n: n}); err != nil {
			z.logger.LogError(z.ctx, "Error occurred while advertising node join.", logwrap.Err(err), logwrap.Datum("Identifier", d.address.String()))
		}
	}
}

func (z *ZDA) receiveNodeLeaveEvent(e zigbee.NodeLeaveEvent) {
	z.logger.LogInfo(z.ctx, "Node has left zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

	if n := z.getNode(e.IEEEAddress); n != nil {
		for _, d := range z.getDevicesOnNode(n) {
			_ = z.logger.SegmentFn(z.ctx, "Device leaving zigbee network.", logwrap.Datum("Identifier", d.address.String()))(func(ctx context.Context) error {
				z.logger.LogInfo(ctx, "Remove device upon node leaving zigbee network.")
				_ = z.removeDevice(ctx, d.address)
				return nil
			})
		}

		_ = z.removeNode(e.IEEEAddress)
	} else {
		z.logger.LogWarn(z.ctx, "Receive leave message for unknown node from provider.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))
	}
}

func (z *ZDA) receiveNodeIncomingMessageEvent(e zigbee.NodeIncomingMessageEvent) {
	if err := z.zclCommunicator.ProcessIncomingMessage(e); err != nil {
		z.logger.LogWarn(z.ctx, "ZCL communicator failed to process incoming message.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()), logwrap.Err(err))
		return
	}
}
