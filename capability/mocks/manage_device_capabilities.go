package mocks

import (
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/zda/capability"
	"github.com/stretchr/testify/mock"
)

type MockManageDeviceCapabilities struct {
	mock.Mock
}

func (m *MockManageDeviceCapabilities) Add(d Device, c da.Capability) {
	m.Called(d, c)
}

func (m *MockManageDeviceCapabilities) Remove(d Device, c da.Capability) {
	m.Called(d, c)
}
