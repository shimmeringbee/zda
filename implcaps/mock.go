package implcaps

import (
	"github.com/shimmeringbee/zda/attribute"
	"github.com/stretchr/testify/mock"
)

type MockZDAInterface struct {
	mock.Mock
}

func (m *MockZDAInterface) NewAttributeMonitor() attribute.Monitor {
	return m.Called().Get(0).(attribute.Monitor)
}

func (m *MockZDAInterface) SendEvent(a any) {
	m.Called(a)
}

var _ ZDAInterface = (*MockZDAInterface)(nil)