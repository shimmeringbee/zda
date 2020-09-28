package mocks

import (
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/zda/capability"
	"github.com/stretchr/testify/mock"
)

type MockComposeDADevice struct {
	mock.Mock
}

func (m *MockComposeDADevice) Compose(c Device) da.Device {
	ret := m.Called(c)
	return ret.Get(0).(da.Device)
}
