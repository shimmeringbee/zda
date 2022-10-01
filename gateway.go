package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
)

func New(p zigbee.Provider) da.Gateway {
	return &gateway{
		provider: p,
	}
}

type gateway struct {
	provider zigbee.Provider

	selfDevice da.BaseDevice
}

func (g *gateway) ReadEvent(_ context.Context) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) Capability(_ da.Capability) interface{} {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) Capabilities() []da.Capability {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) Self() da.Device {
	return g.selfDevice
}

func (g *gateway) Devices() []da.Device {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) Start() error {
	adapterNode := g.provider.AdapterNode()

	g.selfDevice = da.BaseDevice{
		DeviceGateway:      g,
		DeviceIdentifier:   adapterNode.IEEEAddress,
		DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag},
	}

	return nil
}

func (g *gateway) Stop() error {
	return nil
}

var _ da.Gateway = (*gateway)(nil)
