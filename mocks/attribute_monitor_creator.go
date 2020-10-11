package mocks

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockAttributeMonitorCreator struct {
	mock.Mock
}

func (m *MockAttributeMonitorCreator) Create(bc zda.BasicCapability, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, cb func(zda.Device, zcl.AttributeID, zcl.AttributeDataTypeValue)) zda.AttributeMonitor {
	args := m.Called(bc, c, a, dt, cb)
	return args.Get(0).(zda.AttributeMonitor)
}

type MockAttributeMonitor struct {
	mock.Mock
}

func (m *MockAttributeMonitor) Attach(c context.Context, d zda.Device, e zigbee.Endpoint, v interface{}) (bool, error) {
	args := m.Called(c, d, e, v)
	return args.Bool(0), args.Error(1)
}

func (m *MockAttributeMonitor) Detach(c context.Context, d zda.Device) {
	m.Called(c, d)
}

func (m *MockAttributeMonitor) Load(c context.Context, d zda.Device, e zigbee.Endpoint, b bool) {
	m.Called(c, d, e, b)
}

func (m *MockAttributeMonitor) Poll(c context.Context, d zda.Device) {
	m.Called(c, d)
}
