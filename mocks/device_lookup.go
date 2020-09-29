package mocks

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zda"
	"github.com/stretchr/testify/mock"
)

type MockDeviceLookup struct {
	mock.Mock
}

func (m *MockDeviceLookup) ByDA(d da.Device) (zda.Device, bool) {
	ret := m.Called(d)
	return ret.Get(0).(zda.Device), ret.Bool(1)
}
