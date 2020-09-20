package zda

import (
	"context"
	"errors"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"log"
	"sync"
	"time"
)

const DefaultGatewayHomeAutomationEndpoint = zigbee.Endpoint(0x01)

type ZigbeeGateway struct {
	provider     zigbee.Provider
	communicator *communicator.Communicator

	selfNode *internalNode
	self     *internalDevice

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
	poller    *zdaPoller
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)
	onoff.Register(zclCommandRegistry)

	zgw := &ZigbeeGateway{
		provider:     provider,
		communicator: communicator.NewCommunicator(provider, zclCommandRegistry),

		selfNode: &internalNode{mutex: &sync.RWMutex{}},
		self:     &internalDevice{mutex: &sync.RWMutex{}},

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

	zgw.poller = &zdaPoller{nodeStore: zgw}

	zgw.capabilities[DeviceDiscoveryFlag] = &ZigbeeDeviceDiscovery{
		gateway:        zgw,
		networkJoining: zgw.provider,
		eventSender:    zgw,
	}

	zgw.capabilities[EnumerateDeviceFlag] = &ZigbeeEnumerateDevice{
		gateway:           zgw,
		deviceStore:       zgw,
		eventSender:       zgw,
		nodeQuerier:       zgw.provider,
		internalCallbacks: zgw.callbacks,
	}

	zgw.capabilities[LocalDebugFlag] = &ZigbeeLocalDebug{gateway: zgw}

	zgw.capabilities[HasProductInformationFlag] = &ZigbeeHasProductInformation{
		gateway:               zgw,
		deviceStore:           zgw,
		internalCallbacks:     zgw.callbacks,
		zclGlobalCommunicator: zgw.communicator.Global(),
	}

	zgw.capabilities[OnOffFlag] = &ZigbeeOnOff{
		gateway:                  zgw,
		internalCallbacks:        zgw.callbacks,
		deviceStore:              zgw,
		nodeStore:                zgw,
		zclCommunicatorCallbacks: zgw.communicator,
		zclCommunicatorRequests:  zgw.communicator,
		zclGlobalCommunicator:    zgw.communicator.Global(),
		nodeBinder:               zgw.provider,
		poller:                   zgw.poller,
		eventSender:              zgw,
	}

	initOrder := []Capability{
		DeviceDiscoveryFlag,
		EnumerateDeviceFlag,
		LocalDebugFlag,
		HasProductInformationFlag,
		OnOffFlag,
	}

	for _, capability := range initOrder {
		capabilityImpl := zgw.capabilities[capability]

		if initable, is := capabilityImpl.(CapabilityInitable); is {
			initable.Init()
		}
	}

	zgw.callbacks.Add(zgw.enableAPSACK)

	return zgw
}

func (z *ZigbeeGateway) Start() error {
	z.selfNode.ieeeAddress = z.provider.AdapterNode().IEEEAddress

	z.self.node = z.selfNode
	z.self.subidentifier = 0
	z.self.capabilities = []Capability{
		DeviceDiscoveryFlag,
	}

	if err := z.provider.RegisterAdapterEndpoint(z.context, DefaultGatewayHomeAutomationEndpoint, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		return err
	}

	z.poller.Start()

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

	z.poller.Stop()

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

		if err != nil && !errors.Is(err, zigbee.ContextExpired) {
			log.Printf("could not listen for event from zigbee provider: %+v", err)
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			iNode, found := z.getNode(e.IEEEAddress)

			if !found {
				iNode = z.addNode(e.IEEEAddress)
			}

			if len(iNode.getDevices()) == 0 {
				initialDeviceId := iNode.nextDeviceIdentifier()

				z.addDevice(initialDeviceId, iNode)

				z.callbacks.Call(context.Background(), internalNodeJoin{node: iNode})
			}

		case zigbee.NodeLeaveEvent:
			iNode, found := z.getNode(e.IEEEAddress)

			if found {
				z.callbacks.Call(context.Background(), internalNodeLeave{node: iNode})

				for _, iDev := range iNode.getDevices() {
					z.removeDevice(iDev.generateIdentifier())
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
	return z.self.toDevice(z)
}

func (z *ZigbeeGateway) Devices() []Device {
	devices := []Device{z.Self()}

	z.nodesLock.RLock()

	for _, iNode := range z.nodes {
		iNode.mutex.RLock()

		for _, iDev := range iNode.devices {
			devices = append(devices, iDev.toDevice(z))
		}

		iNode.mutex.RUnlock()
	}

	z.nodesLock.RUnlock()

	return devices
}
