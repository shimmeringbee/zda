package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockDAEventSender struct {
	mock.Mock
}

func (m *MockDAEventSender) Send(i interface{}) {
	m.Called(i)
}
