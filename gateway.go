package zda

import (
	"context"
	"errors"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
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

	events chan interface{}

	nodeTable nodeTable

	callbacks *callbacks.Callbacks
	poller    *zdaPoller

	capabilityManager *CapabilityManager
}

func New(p zigbee.Provider) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)
	onoff.Register(zclCommandRegistry)

	callbacker := callbacks.Create()

	nodeTable := newNodeTable()
	nodeTable.callbacks = callbacker

	zgw := &ZigbeeGateway{
		provider:     p,
		communicator: communicator.NewCommunicator(p, zclCommandRegistry),

		selfNode: &internalNode{mutex: &sync.RWMutex{}},
		self:     &internalDevice{mutex: &sync.RWMutex{}},

		providerHandlerStop: make(chan bool, 1),
		context:             ctx,
		contextCancel:       cancel,

		events: make(chan interface{}, 100),

		nodeTable: nodeTable,
		callbacks: callbacker,

		capabilityManager: NewCapabilityManager(),
	}

	zgw.poller = &zdaPoller{nodeTable: zgw.nodeTable}

	zgw.callbacks.Add(zgw.enableAPSACK)

	zgw.capabilityManager.Add(&ZigbeeDeviceDiscovery{})
	zgw.capabilityManager.Add(&ZigbeeEnumerateDevice{})

	zgw.capabilityManager.Init()

	return zgw
}

func (z *ZigbeeGateway) Start() error {
	z.selfNode.ieeeAddress = z.provider.AdapterNode().IEEEAddress

	z.self.node = z.selfNode
	z.self.subidentifier = 0
	z.self.capabilities = []da.Capability{
		capabilities.DeviceDiscoveryFlag,
	}

	if err := z.provider.RegisterAdapterEndpoint(z.context, DefaultGatewayHomeAutomationEndpoint, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		return err
	}

	z.callbacks.Add(func(ctx context.Context, e internalDeviceAdded) error {
		z.sendEvent(da.DeviceAdded{Device: e.device.toDevice(z)})
		return nil
	})

	z.callbacks.Add(func(ctx context.Context, e internalDeviceRemoved) error {
		z.sendEvent(da.DeviceRemoved{Device: e.device.toDevice(z)})
		return nil
	})

	z.poller.Start()

	go z.providerHandler()

	z.capabilityManager.Start()

	return nil
}

func (z *ZigbeeGateway) Stop() error {
	z.providerHandlerStop <- true
	z.contextCancel()

	z.poller.Stop()

	z.capabilityManager.Stop()

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

func (z *ZigbeeGateway) sendEvent(e interface{}) {
	select {
	case z.events <- e:
	default:
		fmt.Printf("warning could not send event, channel buffer full: %+v", e)
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

func (z *ZigbeeGateway) Capability(c da.Capability) interface{} {
	return z.capabilityManager.Get(c)
}

func (z *ZigbeeGateway) Self() da.Device {
	return z.self.toDevice(z)
}

func (z *ZigbeeGateway) Devices() []da.Device {
	devices := []da.Device{z.Self()}

	for _, iDev := range z.nodeTable.getDevices() {
		devices = append(devices, iDev.toDevice(z))
	}

	return devices
}
