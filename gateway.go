package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
	"log"
	"os"
)

func New(baseCtx context.Context, p zigbee.Provider) da.Gateway {
	gw := &gateway{ctx: baseCtx, provider: p}
	gw.WithGoLogger(log.New(os.Stderr, "", log.LstdFlags))
	return gw
}

type gateway struct {
	provider zigbee.Provider

	logger logwrap.Logger
	ctx    context.Context

	selfDevice da.SimpleDevice
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

func (g *gateway) Start(_ context.Context) error {
	g.logger.LogInfo(g.ctx, "Starting ZDA.")

	adapterNode := g.provider.AdapterNode()

	g.selfDevice = da.SimpleDevice{
		DeviceGateway:      g,
		DeviceIdentifier:   adapterNode.IEEEAddress,
		DeviceCapabilities: []da.Capability{capabilities.DeviceDiscoveryFlag},
	}

	g.logger.LogInfo(g.ctx, "Adapter coordinator IEEE address.", logwrap.Datum("IEEEAddress", g.selfDevice.Identifier().String()))

	return nil
}

func (g *gateway) Stop(_ context.Context) error {
	return nil
}

var _ da.Gateway = (*gateway)(nil)
