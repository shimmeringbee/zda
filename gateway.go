package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
)

func New(p zigbee.Provider) *da.Gateway {
	//TODO implement me
	panic("implement me")
}

type gateway struct {
}

func (g gateway) ReadEvent(ctx context.Context) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Capability(c da.Capability) interface{} {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Capabilities() []da.Capability {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Self() da.Device {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Devices() []da.Device {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Start() error {
	//TODO implement me
	panic("implement me")
}

func (g gateway) Stop() error {
	//TODO implement me
	panic("implement me")
}

var _ da.Gateway = (*gateway)(nil)
