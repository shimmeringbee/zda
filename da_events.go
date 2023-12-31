package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
)

type eventSender interface {
	sendEvent(event interface{})
}

func (g *gateway) sendEvent(e interface{}) {
	select {
	case g.events <- e:
	default:
		g.logger.LogError(g.ctx, "failed to send event, channel buffer is full", logwrap.Datum("event", e))
	}
}

func (g *gateway) ReadEvent(ctx context.Context) (interface{}, error) {
	select {
	case e := <-g.events:
		return e, nil
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	}
}
