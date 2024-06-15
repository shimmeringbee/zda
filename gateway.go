package zda

import (
	"context"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/factory"
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"log"
	"os"
	"sync"
)

const DefaultGatewayHomeAutomationEndpoint = zigbee.Endpoint(0x01)

func New(baseCtx context.Context, s persistence.Section, p zigbee.Provider, r ruleExecutor) *ZDA {
	ctx, cancel := context.WithCancel(baseCtx)

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)

	gw := &ZDA{
		provider:           p,
		zclCommunicator:    communicator.NewCommunicator(p, zclCommandRegistry),
		zclCommandRegistry: zclCommandRegistry,

		selfDevice: gatewayDevice{
			dd: &deviceDiscovery{},
		},

		ctx:       ctx,
		ctxCancel: cancel,

		nodeLock: &sync.RWMutex{},
		node:     make(map[zigbee.IEEEAddress]*node),

		section: s,

		callbacks:    callbacks.Create(),
		ruleExecutor: r,

		events: make(chan any, 1),
	}

	gw.zdaInterface = zdaInterface{
		gw: gw,
		c:  gw.zclCommunicator,
	}

	gw.WithGoLogger(log.New(os.Stderr, "", log.LstdFlags))

	gw.ed = &enumerateDevice{
		gw:                gw,
		dm:                gw,
		logger:            gw.logger,
		nq:                gw.provider,
		zclReadFn:         gw.zclCommunicator.ReadAttributes,
		capabilityFactory: factory.Create,
		es:                gw,
	}

	if gw.ruleExecutor != nil {
		gw.ed.runRulesFn = gw.ruleExecutor.Execute
	}

	gw.callbacks.Add(gw.ed.onNodeJoin)

	return gw
}

type ruleExecutor interface {
	Execute(rules.Input) (rules.Output, error)
}

type ZDA struct {
	provider        zigbee.Provider
	zclCommunicator communicator.Communicator
	zdaInterface    implcaps.ZDAInterface

	logger    logwrap.Logger
	ctx       context.Context
	ctxCancel func()

	selfDevice gatewayDevice

	nodeLock *sync.RWMutex
	node     map[zigbee.IEEEAddress]*node

	section persistence.Section

	callbacks    callbacks.AdderCaller
	ruleExecutor ruleExecutor

	ed                 *enumerateDevice
	events             chan any
	zclCommandRegistry *zcl.CommandRegistry
}

func (z *ZDA) Capabilities() []da.Capability {
	caps := map[da.Capability]struct{}{capabilities.DeviceRemovalFlag: {}, capabilities.EnumerateDeviceFlag: {}}

	for _, c := range factory.Mapping {
		caps[c] = struct{}{}
	}

	var capSlice []da.Capability

	for c := range caps {
		capSlice = append(capSlice, c)
	}

	return capSlice
}

func (z *ZDA) Self() da.Device {
	return z.selfDevice
}

func (z *ZDA) Devices() []da.Device {
	allDevices := []da.Device{z.Self()}

	for _, d := range z.getDevices() {
		allDevices = append(allDevices, d)
	}

	return allDevices
}

func (z *ZDA) Start(ctx context.Context) error {
	z.logger.LogInfo(z.ctx, "Starting ZDA.")

	adapterNode := z.provider.AdapterNode()

	z.selfDevice = gatewayDevice{
		gateway:    z,
		identifier: adapterNode.IEEEAddress,
		dd: &deviceDiscovery{
			gateway:        z,
			networkJoining: z.provider,
			eventSender:    z,
			logger:         z.logger,
		},
	}

	z.logger.LogInfo(z.ctx, "Adapter coordinator IEEE address.", logwrap.Datum("IEEEAddress", z.selfDevice.Identifier().String()))

	if err := z.provider.RegisterAdapterEndpoint(ctx, DefaultGatewayHomeAutomationEndpoint, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		z.logger.LogError(z.ctx, "Failed to register endpoint against adapter.", logwrap.Datum("Endpoint", DefaultGatewayHomeAutomationEndpoint), logwrap.Err(err))
		return err
	}

	go z.providerLoop()

	return nil
}

func (z *ZDA) Stop(_ context.Context) error {
	z.logger.LogInfo(z.ctx, "Stopping ZDA.")
	z.selfDevice.dd.Stop()
	z.ctxCancel()
	return nil
}

var _ da.Gateway = (*ZDA)(nil)

type gatewayDevice struct {
	gateway    da.Gateway
	identifier da.Identifier
	dd         *deviceDiscovery
}

func (g gatewayDevice) Gateway() da.Gateway {
	return g.gateway
}

func (g gatewayDevice) Identifier() da.Identifier {
	return g.identifier
}

func (g gatewayDevice) Capabilities() []da.Capability {
	return []da.Capability{capabilities.DeviceDiscoveryFlag}
}

func (g gatewayDevice) Capability(capability da.Capability) da.BasicCapability {
	switch capability {
	case capabilities.DeviceDiscoveryFlag:
		return g.dd
	default:
		return nil
	}
}

var _ da.Device = (*gatewayDevice)(nil)
