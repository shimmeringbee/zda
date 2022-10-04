package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/logwrap"
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
	}
}

func (g *gateway) receiveNodeLeaveEvent(e zigbee.NodeLeaveEvent) {
	g.logger.LogInfo(g.ctx, "Node has left zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

	if n := g.getNode(e.IEEEAddress); n != nil {
		for _, d := range g.getDevicesOnNode(n) {
			g.logger.LogInfo(g.ctx, "Remove device upon node leaving zigbee network.", logwrap.Datum("Identifier", d.address.String()))
			g.removeDevice(d.address)
		}

		_ = g.removeNode(e.IEEEAddress)
	} else {
		g.logger.LogWarn(g.ctx, "Receive leave message for unknown node from provider.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))
	}
}

func (g *gateway) receiveNodeIncomingMessageEvent(e zigbee.NodeIncomingMessageEvent) {

}
