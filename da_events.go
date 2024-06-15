package zda

import (
	"context"
)

type eventSender interface {
	sendEvent(event any)
}

func (z *ZDA) sendEvent(e any) {
	z.events <- e
}

func (z *ZDA) ReadEvent(ctx context.Context) (any, error) {
	select {
	case e := <-z.events:
		return e, nil
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	}
}
