package attribute

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
)

type MonitorCallback func(zcl.AttributeID, zcl.AttributeDataTypeValue)

type Monitor interface {
	Init(s persistence.Section, d da.Device, cb MonitorCallback)
	Load(ctx context.Context) error
	Attach(ctx context.Context, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, forcePolling bool) error
	Detach(ctx context.Context, unconfigure bool) error
}

func NewMonitor(c communicator.Communicator) Monitor {
	return &zclMonitor{}
}

type zclMonitor struct {
}

func (z zclMonitor) Init(s persistence.Section, d da.Device, cb MonitorCallback) {
	//TODO implement me
	panic("implement me")
}

func (z zclMonitor) Load(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (z zclMonitor) Attach(ctx context.Context, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, forcePolling bool) error {
	//TODO implement me
	panic("implement me")
}

func (z zclMonitor) Detach(ctx context.Context, unconfigure bool) error {
	//TODO implement me
	panic("implement me")
}
