package zda

import (
	"context"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
)

type ZigbeeGateway struct {
	provider zigbee.Provider
	self     Device

	providerHandlerStop chan bool
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	return &ZigbeeGateway{
		provider: provider,
		self: Device{
			Capabilities: []Capability{
				DeviceDiscoveryFlag,
			},
		},
		providerHandlerStop: make(chan bool, 1),
	}
}

func (z *ZigbeeGateway) Start() error {
	z.self.Gateway = z
	z.self.Identifier = z.provider.AdapterNode().IEEEAddress
	z.self.Capabilities = []Capability{
		DeviceDiscoveryFlag,
	}

	go z.providerHandler()
	return nil
}

func (z *ZigbeeGateway) Stop() error {
	z.providerHandlerStop <- true
	return nil
}

func (z *ZigbeeGateway) providerHandler() {
	for {
		select {
		case <-z.providerHandlerStop:
			return
		}
	}
}

func (z *ZigbeeGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (z *ZigbeeGateway) Capability(capability Capability) interface{} {
	return nil
}

func (z *ZigbeeGateway) Self() Device {
	return z.self
}

func (z *ZigbeeGateway) Devices() []Device {
	return []Device{z.self}
}
