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
				g.logger.LogInfo(g.ctx, "Provider loop terminating due to cancelled context.", logwrap.Err(err))
			} else {
				g.logger.LogError(g.ctx, "Failed to read event from Zigbee provider.", logwrap.Err(err))
			}
			return
		}

		switch event.(type) {
		case zigbee.NodeJoinEvent:
		case zigbee.NodeUpdateEvent:
		case zigbee.NodeLeaveEvent:
		case zigbee.NodeIncomingMessageEvent:
		}
	}
}
