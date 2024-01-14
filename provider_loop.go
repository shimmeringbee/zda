package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
)

func (g *gateway) providerLoop() {
	for {
		event, err := g.provider.ReadEvent(g.ctx)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				g.logger.LogInfo(g.ctx, "Provider loop terminating due to cancelled context.")
			} else {
				g.logger.LogError(g.ctx, "Failed to read event from Zigbee provider.", logwrap.Err(err))
			}
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			g.receiveNodeJoinEvent(e)
		case zigbee.NodeLeaveEvent:
			g.receiveNodeLeaveEvent(e)
		case zigbee.NodeIncomingMessageEvent:
			g.receiveNodeIncomingMessageEvent(e)
		}
	}
}

func (g *gateway) receiveNodeJoinEvent(e zigbee.NodeJoinEvent) {
	g.logger.LogInfo(g.ctx, "Node has joined zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

	if n, created := g.createNode(e.IEEEAddress); created {
		d := g.createNextDevice(n)
		g.logger.LogInfo(g.ctx, "Created default device.", logwrap.Datum("Identifier", d.address.String()))

		if err := g.callbacks.Call(g.ctx, nodeJoin{n: n}); err != nil {
			g.logger.LogError(g.ctx, "Error occurred while advertising node join.", logwrap.Err(err), logwrap.Datum("Identifier", d.address.String()))
		}
	}
}

func (g *gateway) receiveNodeLeaveEvent(e zigbee.NodeLeaveEvent) {
	g.logger.LogInfo(g.ctx, "Node has left zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

	if n := g.getNode(e.IEEEAddress); n != nil {
		for _, d := range g.getDevicesOnNode(n) {
			ctx, end := g.logger.Segment(g.ctx, "Device leaving zigbee network.", logwrap.Datum("Identifier", d.address.String()))

			g.logger.LogInfo(ctx, "Remove device upon node leaving zigbee network.")
			_ = g.removeDevice(d.address)
			for cf, impl := range d.capabilities {
				g.logger.LogInfo(ctx, "Detaching capability from removed device.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()))
				if err := impl.Detach(ctx, implcaps.DeviceRemoved); err != nil {
					g.logger.LogWarn(ctx, "Error thrown while detaching capability.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()), logwrap.Err(err))
				}
			}

			end()
		}

		_ = g.removeNode(e.IEEEAddress)
	} else {
		g.logger.LogWarn(g.ctx, "Receive leave message for unknown node from provider.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))
	}
}

func (g *gateway) receiveNodeIncomingMessageEvent(e zigbee.NodeIncomingMessageEvent) {
	if err := g.zclCommunicator.ProcessIncomingMessage(e); err != nil {
		g.logger.LogWarn(g.ctx, "ZCL communicator failed to process incoming message.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()), logwrap.Err(err))
		return
	}
}
