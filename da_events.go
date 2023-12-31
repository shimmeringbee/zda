package zda

import "context"

type eventSender interface {
	sendEvent(event interface{})
}

func (g *gateway) sendEvent(event interface{}) {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) ReadEvent(_ context.Context) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}
