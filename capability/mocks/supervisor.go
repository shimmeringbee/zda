package mocks

import (
	. "github.com/shimmeringbee/zda/capability"
	"github.com/stretchr/testify/mock"
)

type MockSupervisor struct {
	mock.Mock
}

func (m *MockSupervisor) FetchCapability() FetchCapability {
	ret := m.Called()
	return ret.Get(0).(FetchCapability)
}

func (m *MockSupervisor) ManageDeviceCapabilities() ManageDeviceCapabilities {
	ret := m.Called()
	return ret.Get(0).(ManageDeviceCapabilities)
}

func (m *MockSupervisor) EventSubscription() EventSubscription {
	ret := m.Called()
	return ret.Get(0).(EventSubscription)
}

func (m *MockSupervisor) ComposeDADevice() ComposeDADevice {
	ret := m.Called()
	return ret.Get(0).(ComposeDADevice)
}

func (m *MockSupervisor) DeviceLookup() DeviceLookup {
	ret := m.Called()
	return ret.Get(0).(DeviceLookup)
}

func (m *MockSupervisor) ZCL() ZCL {
	ret := m.Called()
	return ret.Get(0).(ZCL)
}