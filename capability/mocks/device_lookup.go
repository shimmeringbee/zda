package mocks

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zda/capability"
	"github.com/stretchr/testify/mock"
)

type MockDeviceLookup struct {
	mock.Mock
}

func (m *MockDeviceLookup) ByDA(d da.Device) (capability.Device, bool) {
	ret := m.Called(d)
	return ret.Get(0).(capability.Device), ret.Bool(1)
}
