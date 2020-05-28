package zda

import (
	"context"
	"errors"
	"fmt"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"log"
	"sync"
	"time"
)

type ZigbeeGateway struct {
	provider zigbee.Provider
	self     ZigbeeDevice

	context             context.Context
	contextCancel       context.CancelFunc
	providerHandlerStop chan bool

	events       chan interface{}
	capabilities map[Capability]interface{}

	devices    map[Identifier]*ZigbeeDevice
	deviceLock *sync.RWMutex
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zgw := &ZigbeeGateway{
		provider: provider,
		self:     ZigbeeDevice{mutex: &sync.RWMutex{}},

		providerHandlerStop: make(chan bool, 1),
		context:             ctx,
		contextCancel:       cancel,

		events:       make(chan interface{}, 100),
		capabilities: map[Capability]interface{}{},

		devices:    map[Identifier]*ZigbeeDevice{},
		deviceLock: &sync.RWMutex{},
	}

	zgw.capabilities[DeviceDiscoveryFlag] = &ZigbeeDeviceDiscovery{gateway: zgw}
	zgw.capabilities[EnumerateDeviceFlag] = &ZigbeeEnumerateDevice{gateway: zgw}

	return zgw
}

func (z *ZigbeeGateway) Start() error {
	z.self.device.Gateway = z
	z.self.device.Identifier = z.provider.AdapterNode().IEEEAddress
	z.self.device.Capabilities = []Capability{
		DeviceDiscoveryFlag,
	}

	go z.providerHandler()
	z.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice).Start()
	return nil
}

func (z *ZigbeeGateway) Stop() error {
	z.providerHandlerStop <- true
	z.contextCancel()
	z.capabilities[DeviceDiscoveryFlag].(*ZigbeeDeviceDiscovery).Stop()
	z.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice).Stop()
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

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			_, found := z.getDevice(e.IEEEAddress)

			if !found {
				zDevice := z.addDevice(e.IEEEAddress)
				z.sendEvent(DeviceAdded{Device: zDevice.device})
			}

		case zigbee.NodeLeaveEvent:
			zDevice, found := z.getDevice(e.IEEEAddress)

			if found {
				z.removeDevice(e.IEEEAddress)
				z.sendEvent(DeviceRemoved{Device: zDevice.device})
			}
		}

		select {
		case <-z.providerHandlerStop:
			return
		default:
		}
	}
}

func (z *ZigbeeGateway) getDevice(identifier Identifier) (*ZigbeeDevice, bool) {
	z.deviceLock.RLock()
	defer z.deviceLock.RUnlock()

	device, found := z.devices[identifier]
	return device, found
}

func (z *ZigbeeGateway) addDevice(identifier Identifier) *ZigbeeDevice {
	z.deviceLock.Lock()
	defer z.deviceLock.Unlock()

	device := Device{
		Gateway:      z,
		Identifier:   identifier,
		Capabilities: []Capability{EnumerateDeviceFlag},
	}

	z.devices[identifier] = &ZigbeeDevice{
		device:               device,
		mutex:                &sync.RWMutex{},
		endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
	}

	return z.devices[identifier]
}

func (z *ZigbeeGateway) removeDevice(identifier Identifier) {
	z.deviceLock.Lock()
	defer z.deviceLock.Unlock()

	delete(z.devices, identifier)
}

func (z *ZigbeeGateway) sendEvent(event interface{}) {
	select {
	case z.events <- event:
	default:
		fmt.Printf("warning could not send event, channel buffer full: %+v", event)
	}
}

func (z *ZigbeeGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	select {
	case event := <-z.events:
		return event, nil
	case <-ctx.Done():
		return nil, errors.New("context expired")
	}
}

func (z *ZigbeeGateway) Capability(capability Capability) interface{} {
	return z.capabilities[capability]
}

func (z *ZigbeeGateway) Self() Device {
	return z.self.device
}

func (z *ZigbeeGateway) Devices() []Device {
	return []Device{z.self.device}
}
