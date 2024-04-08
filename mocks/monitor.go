package mocks

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockMonitor struct {
	mock.Mock
}

func (m *MockMonitor) Init(s persistence.Section, d da.Device, cb attribute.MonitorCallback) {
	m.Called(s, d, cb)
}

func (m *MockMonitor) Load(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockMonitor) Attach(ctx context.Context, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, forcePolling bool) error {
	return m.Called(ctx, e, c, a, forcePolling).Error(0)
}

func (m *MockMonitor) Detach(ctx context.Context, unconfigure bool) error {
	return m.Called(ctx, unconfigure).Error(0)

}

var _ attribute.Monitor = (*MockMonitor)(nil)
