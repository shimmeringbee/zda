package zda

import (
	"context"
)

type eventSender interface {
	sendEvent(event interface{})
}

func (g *gateway) sendEvent(e interface{}) {
	g.events <- e
}

func (g *gateway) ReadEvent(ctx context.Context) (interface{}, error) {
	select {
	case e := <-g.events:
		return e, nil
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	}
}
