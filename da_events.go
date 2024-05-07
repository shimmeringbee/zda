package zda

import (
	"context"
)

type eventSender interface {
	sendEvent(event any)
}

func (g *gateway) sendEvent(e any) {
	g.events <- e
}

func (g *gateway) ReadEvent(ctx context.Context) (any, error) {
	select {
	case e := <-g.events:
		return e, nil
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	}
}
