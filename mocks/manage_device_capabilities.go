package mocks

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zda"
	"github.com/stretchr/testify/mock"
)

type MockManageDeviceCapabilities struct {
	mock.Mock
}

func (m *MockManageDeviceCapabilities) Add(d zda.Device, c da.Capability) {
	m.Called(d, c)
}

func (m *MockManageDeviceCapabilities) Remove(d zda.Device, c da.Capability) {
	m.Called(d, c)
}
