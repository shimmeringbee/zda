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

	nodeTable nodeTable

	callbacks *callbacks.Callbacks
	poller    *zdaPoller
}

func New(provider zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)
	onoff.Register(zclCommandRegistry)

	callbacker := callbacks.Create()

	nodeTable := newNodeTable()
	nodeTable.callbacks = callbacker

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

		nodeTable: nodeTable,
		callbacks: callbacker,
	}

	zgw.poller = &zdaPoller{nodeTable: zgw.nodeTable}

	zgw.capabilities[DeviceDiscoveryFlag] = &ZigbeeDeviceDiscovery{
		gateway:        zgw,
		networkJoining: zgw.provider,
		eventSender:    zgw,
	}

	zgw.capabilities[EnumerateDeviceFlag] = &ZigbeeEnumerateDevice{
		gateway:           zgw,
		nodeTable:         zgw.nodeTable,
		eventSender:       zgw,
		nodeQuerier:       zgw.provider,
		internalCallbacks: zgw.callbacks,
	}

	zgw.capabilities[LocalDebugFlag] = &ZigbeeLocalDebug{
		gateway:   zgw,
		nodeTable: zgw.nodeTable,
	}

	zgw.capabilities[HasProductInformationFlag] = &ZigbeeHasProductInformation{
		gateway:               zgw,
		nodeTable:             zgw.nodeTable,
		internalCallbacks:     zgw.callbacks,
		zclGlobalCommunicator: zgw.communicator.Global(),
	}

	zgw.capabilities[OnOffFlag] = &ZigbeeOnOff{
		gateway:                  zgw,
		internalCallbacks:        zgw.callbacks,
		nodeTable:                zgw.nodeTable,
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

	z.callbacks.Add(func(ctx context.Context, event internalDeviceAdded) error {
		z.sendEvent(DeviceAdded{Device: event.device.toDevice(z)})
		return nil
	})

	z.callbacks.Add(func(ctx context.Context, event internalDeviceRemoved) error {
		z.sendEvent(DeviceRemoved{Device: event.device.toDevice(z)})
		return nil
	})

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
			iNode, _ := z.nodeTable.createNode(e.IEEEAddress)

			if len(iNode.getDevices()) == 0 {
				z.nodeTable.createNextDevice(e.IEEEAddress)

				z.callbacks.Call(context.Background(), internalNodeJoin{node: iNode})
			}

		case zigbee.NodeLeaveEvent:
			iNode := z.nodeTable.getNode(e.IEEEAddress)

			if iNode != nil {
				z.callbacks.Call(context.Background(), internalNodeLeave{node: iNode})

				for _, iDev := range iNode.getDevices() {
					z.nodeTable.removeDevice(iDev.generateIdentifier())
				}

				z.nodeTable.removeNode(e.IEEEAddress)
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

	for _, iDev := range z.nodeTable.getDevices() {
		devices = append(devices, iDev.toDevice(z))
	}

	return devices
}
