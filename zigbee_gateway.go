package zda

import (
	"context"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"log"
	"time"
)

type ZigbeeGateway struct {
	provider zigbee.Provider
	self     Device

	context             context.Context
	contextCancel       context.CancelFunc
	providerHandlerStop chan bool
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	return &ZigbeeGateway{
		provider:            provider,
		self:                Device{},
		providerHandlerStop: make(chan bool, 1),
		context:             ctx,
		contextCancel:       cancel,
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
	z.contextCancel()
	return nil
}

func (z *ZigbeeGateway) providerHandler() {
	for {
		ctx, cancel := context.WithTimeout(z.context, 250*time.Millisecond)
		event, err := z.provider.ReadEvent(ctx)
		cancel()

		if err != nil {
			log.Printf("could not listen for event from zigbee provider: %+v", err)
			return
		}

		switch event.(type) {

		}

		select {
		case <-z.providerHandlerStop:
			return
		default:
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
