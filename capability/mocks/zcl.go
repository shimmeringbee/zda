package mocks

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockZCL struct {
	mock.Mock
}

func (m *MockZCL) ReadAttributes(ctx context.Context, d capability.Device, e zigbee.Endpoint, c zigbee.ClusterID, a []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, d, e, c, a)
	return args.Get(0).(map[zcl.AttributeID]global.ReadAttributeResponseRecord), args.Error(1)
}
