package mocks

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockZCL struct {
	mock.Mock
}

func (m *MockZCL) ReadAttributes(ctx context.Context, d zda.Device, e zigbee.Endpoint, c zigbee.ClusterID, a []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, d, e, c, a)
	return args.Get(0).(map[zcl.AttributeID]global.ReadAttributeResponseRecord), args.Error(1)
}

func (m *MockZCL) Bind(ctx context.Context, d zda.Device, e zigbee.Endpoint, c zigbee.ClusterID) error {
	args := m.Called(ctx, d, e, c)
	return args.Error(0)
}

func (m *MockZCL) ConfigureReporting(ctx context.Context, d zda.Device, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, min uint16, max uint16, chg interface{}) error {
	args := m.Called(ctx, d, e, c, a, dt, min, max, chg)
	return args.Error(0)
}

func (m *MockZCL) Listen(f zda.ZCLFilter, c zda.ZCLCallback) {
	m.Called(f, c)
}

func (m *MockZCL) RegisterCommandLibrary(cl zda.ZCLCommandLibrary) {
	m.Called(cl)
}

func (m *MockZCL) SendCommand(ctx context.Context, d zda.Device, e zigbee.Endpoint, c zigbee.ClusterID, cmd interface{}) error {
	args := m.Called(ctx, d, e, c, cmd)
	return args.Error(0)
}
