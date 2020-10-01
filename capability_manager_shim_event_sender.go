package zda

type daEventSenderShim struct {
	eventSender eventSender
}

func (s *daEventSenderShim) Send(e interface{}) {
	s.eventSender.sendEvent(e)
}
