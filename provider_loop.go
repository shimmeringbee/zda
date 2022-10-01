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
		case zigbee.NodeUpdateEvent:
			g.receiveNodeUpdateEvent(e)
		case zigbee.NodeLeaveEvent:
			g.receiveNodeLeaveEvent(e)
		case zigbee.NodeIncomingMessageEvent:
			g.receiveNodeIncomingMessageEvent(e)
		}
	}
}

func (g *gateway) receiveNodeJoinEvent(e zigbee.NodeJoinEvent) {

}

func (g *gateway) receiveNodeUpdateEvent(e zigbee.NodeUpdateEvent) {

}

func (g *gateway) receiveNodeLeaveEvent(e zigbee.NodeLeaveEvent) {

}

func (g *gateway) receiveNodeIncomingMessageEvent(e zigbee.NodeIncomingMessageEvent) {

}
