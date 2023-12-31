package zda

import "github.com/stretchr/testify/mock"

type mockEventSender struct {
	mock.Mock
}

func (m *mockEventSender) sendEvent(event interface{}) {
	m.Called(event)
}
