package zda

import (
	"context"
	"errors"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"log"
	"os"
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

	CapabilityManager *CapabilityManager

	Logger                  logwrap.Logger
	zigbeeEnumerationDevice *ZigbeeEnumerateDevice
	zigbeeDeviceDiscovery   *ZigbeeDeviceDiscovery
	zigbeeDeviceRemoval     *ZigbeeDeviceRemoval
}

func New(p zigbee.Provider, r *rules.Rule) *ZigbeeGateway {
	ctx, cancel := context.WithCancel(context.Background())

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)

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

		events: make(chan interface{}, 1000),

		nodeTable: nodeTable,
		callbacks: callbacker,
	}

	zgw.poller = &zdaPoller{nodeTable: zgw.nodeTable, randLock: &sync.Mutex{}}

	zgw.CapabilityManager = &CapabilityManager{
		gateway:                  zgw,
		deviceCapabilityManager:  zgw,
		eventSender:              zgw,
		nodeTable:                nodeTable,
		callbackAdder:            callbacker,
		poller:                   zgw.poller,
		commandRegistry:          zclCommandRegistry,
		zclGlobalCommunicator:    zgw.communicator.Global(),
		zigbeeNodeBinder:         zgw.provider,
		zclCommunicatorRequests:  zgw.communicator,
		zclCommunicatorCallbacks: zgw.communicator,
		rules:                    r,

		capabilityByFlag:            map[da.Capability]interface{}{},
		capabilityByKeyName:         map[string]PersistableCapability{},
		deviceManagerCapability:     []DeviceManagementCapability{},
		deviceEnumerationCapability: []DeviceEnumerationCapability{},
	}

	zgw.callbacks.Add(zgw.enableAPSACK)

	/* Add internal capabilities that require privileged access to the gateway. */

	/* Add capability to allow manipulation of network joining state. */
	zgw.zigbeeDeviceDiscovery = &ZigbeeDeviceDiscovery{
		gateway:        zgw,
		networkJoining: zgw.provider,
		eventSender:    zgw,
	}
	zgw.CapabilityManager.Add(zgw.zigbeeDeviceDiscovery)

	/* Add capability to allow enumeration and management of devices on nodes. */
	zgw.zigbeeEnumerationDevice = &ZigbeeEnumerateDevice{
		gateway:           zgw,
		nodeTable:         zgw.nodeTable,
		eventSender:       zgw,
		nodeQuerier:       zgw.provider,
		internalCallbacks: zgw.callbacks,
	}
	zgw.CapabilityManager.Add(zgw.zigbeeEnumerationDevice)

	/* Add capability to allow removal of devices. */
	zgw.zigbeeDeviceRemoval = &ZigbeeDeviceRemoval{
		gateway:     zgw,
		nodeTable:   zgw.nodeTable,
		nodeRemover: zgw.provider,
	}
	zgw.CapabilityManager.Add(zgw.zigbeeDeviceRemoval)

	zgw.WithGoLogger(log.New(os.Stderr, "", log.LstdFlags))

	return zgw
}

func (z *ZigbeeGateway) WithGoLogger(parentLogger *log.Logger) {
	z.WithLogWrapLogger(logwrap.New(golog.Wrap(parentLogger)))
}

func (z *ZigbeeGateway) WithLogWrapLogger(lw logwrap.Logger) {
	z.Logger = lw
	z.zigbeeEnumerationDevice.logger = z.Logger
	z.zigbeeDeviceDiscovery.logger = z.Logger
	z.zigbeeDeviceRemoval.logger = z.Logger
	z.CapabilityManager.logger = z.Logger
}

func (z *ZigbeeGateway) Start() error {
	z.Logger.LogInfo(z.context, "Starting ZDA.")

	z.selfNode.ieeeAddress = z.provider.AdapterNode().IEEEAddress

	z.Logger.LogInfo(z.context, "Adapter coordinator IEEE address.", logwrap.Datum("IEEEAddress", z.selfNode.ieeeAddress.String()))

	z.self.node = z.selfNode
	z.self.subidentifier = 0
	z.self.capabilities = []da.Capability{
		capabilities.DeviceDiscoveryFlag,
	}

	if err := z.provider.RegisterAdapterEndpoint(z.context, DefaultGatewayHomeAutomationEndpoint, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		z.Logger.LogError(z.context, "Failed to register endpoint against adapter.", logwrap.Datum("Endpoint", DefaultGatewayHomeAutomationEndpoint), logwrap.Err(err))
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

	z.Logger.LogInfo(z.context, "Initialising capabilities and starting.")
	z.CapabilityManager.Init()
	z.CapabilityManager.Start()

	return nil
}

func (z *ZigbeeGateway) Stop() error {
	z.Logger.LogInfo(z.context, "Stopping ZDA.")

	z.providerHandlerStop <- true
	z.contextCancel()

	z.poller.Stop()

	z.Logger.LogInfo(z.context, "Stopping capabilities.")
	z.CapabilityManager.Stop()

	return nil
}

func (z *ZigbeeGateway) providerHandler() {
	for {
		ctx, cancel := context.WithTimeout(z.context, 250*time.Millisecond)
		event, err := z.provider.ReadEvent(ctx)
		cancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			z.Logger.LogError(z.context, "Failed to read event from Zigbee provider.", logwrap.Err(err))
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			z.Logger.LogInfo(z.context, "Node has joined zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

			iNode, _ := z.nodeTable.createNode(e.IEEEAddress)

			if len(iNode.getDevices()) == 0 {
				z.nodeTable.createNextDevice(e.IEEEAddress)
			}

			z.callbacks.Call(context.Background(), internalNodeJoin{node: iNode})

		case zigbee.NodeLeaveEvent:
			z.Logger.LogInfo(z.context, "Node has left zigbee network.", logwrap.Datum("IEEEAddress", e.IEEEAddress.String()))

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
	case <-time.After(100 * time.Millisecond):
		fmt.Printf("warning could not send event, channel buffer full: %+v", e)
	}
}

func (z *ZigbeeGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	select {
	case event := <-z.events:
		return event, nil
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	}
}

func (z *ZigbeeGateway) Capability(c da.Capability) interface{} {
	return z.CapabilityManager.Get(c)
}

func (z *ZigbeeGateway) Capabilities() []da.Capability {
	return z.CapabilityManager.Capabilities()
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
