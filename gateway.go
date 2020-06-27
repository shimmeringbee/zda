package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"log"
	"sync"
	"time"
)

type ZigbeeGateway struct {
	provider     zigbee.Provider
	communicator *communicator.Communicator

	self *internalDevice

	context             context.Context
	contextCancel       context.CancelFunc
	providerHandlerStop chan bool

	events       chan interface{}
	capabilities map[Capability]interface{}

	devices     map[Identifier]*internalDevice
	devicesLock *sync.RWMutex

	nodes     map[zigbee.IEEEAddress]*internalNode
	nodesLock *sync.RWMutex

	callbacks *callbacks.Callbacks
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)

	zgw := &ZigbeeGateway{
		provider:     provider,
		communicator: communicator.NewCommunicator(provider, zclCommandRegistry),

		self: &internalDevice{mutex: &sync.RWMutex{}},

		providerHandlerStop: make(chan bool, 1),
		context:             ctx,
		contextCancel:       cancel,

		events:       make(chan interface{}, 100),
		capabilities: map[Capability]interface{}{},

		devices:     map[Identifier]*internalDevice{},
		devicesLock: &sync.RWMutex{},

		nodes:     map[zigbee.IEEEAddress]*internalNode{},
		nodesLock: &sync.RWMutex{},

		callbacks: callbacks.Create(),
	}

	zgw.capabilities[DeviceDiscoveryFlag] = &ZigbeeDeviceDiscovery{gateway: zgw}
	zgw.capabilities[EnumerateDeviceFlag] = &ZigbeeEnumerateDevice{gateway: zgw}
	zgw.capabilities[LocalDebugFlag] = &ZigbeeLocalDebug{gateway: zgw}

	for _, capabilityImpl := range zgw.capabilities {
		if initable, is := capabilityImpl.(CapabilityInitable); is {
			initable.Init()
		}
	}

	return zgw
}

func (z *ZigbeeGateway) Start() error {
	z.self.device.Gateway = z
	z.self.device.Identifier = z.provider.AdapterNode().IEEEAddress
	z.self.device.Capabilities = []Capability{
		DeviceDiscoveryFlag,
	}

	if err := z.provider.RegisterAdapterEndpoint(z.context, 1, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		return err
	}

	go z.providerHandler()

	for _, capabilityImpl := range z.capabilities {
		if startable, is := capabilityImpl.(CapabilityStartable); is {
			startable.Start()
		}
	}

	return nil
}

func (z *ZigbeeGateway) Stop() error {
	z.providerHandlerStop <- true
	z.contextCancel()

	for _, capabilityImpl := range z.capabilities {
		if stopable, is := capabilityImpl.(CapabilityStopable); is {
			stopable.Stop()
		}
	}

	return nil
}

func (z *ZigbeeGateway) providerHandler() {
	for {
		ctx, cancel := context.WithTimeout(z.context, 250*time.Millisecond)
		event, err := z.provider.ReadEvent(ctx)
		cancel()

		if err != nil && err != zigbee.ContextExpired {
			log.Printf("could not listen for event from zigbee provider: %+v", err)
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			iNode, found := z.getNode(e.IEEEAddress)

			if !found {
				iNode = z.addNode(e.IEEEAddress)
			}

			initialDeviceId := IEEEAddressWithEndpoint{IEEEAddress: e.IEEEAddress, Endpoint: 0x00}

			_, found = z.getDevice(initialDeviceId)

			if !found {
				iDev := z.addDevice(initialDeviceId, iNode)
				z.sendEvent(DeviceAdded{Device: iDev.device})

				z.callbacks.Call(context.Background(), internalNodeJoin{node: iNode})
			}

		case zigbee.NodeLeaveEvent:
			iNode, found := z.getNode(e.IEEEAddress)

			if found {
				z.callbacks.Call(context.Background(), internalNodeLeave{node: iNode})

				for _, iDev := range iNode.getDevices() {
					z.removeDevice(iDev.device.Identifier)
					z.sendEvent(DeviceRemoved{Device: iDev.device})
				}

				z.removeNode(e.IEEEAddress)
			}

		case zigbee.NodeIncomingMessageEvent:
			z.communicator.ProcessIncomingMessage(e)
		}

		select {
		case <-z.providerHandlerStop:
			return
		default:
		}
	}
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
		return nil, zigbee.ContextExpired
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
