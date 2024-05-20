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

func New(baseCtx context.Context, s persistence.Section, p zigbee.Provider, r ruleExecutor) da.Gateway {
	ctx, cancel := context.WithCancel(baseCtx)

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)

	gw := &gateway{
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

type gateway struct {
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

func (g *gateway) Capabilities() []da.Capability {
	//TODO implement me
	panic("implement me")
}

func (g *gateway) Self() da.Device {
	return g.selfDevice
}

func (g *gateway) Devices() []da.Device {
	allDevices := []da.Device{g.Self()}

	for _, d := range g.getDevices() {
		allDevices = append(allDevices, d)
	}

	return allDevices
}

func (g *gateway) Start(ctx context.Context) error {
	g.logger.LogInfo(g.ctx, "Starting ZDA.")

	adapterNode := g.provider.AdapterNode()

	g.selfDevice = gatewayDevice{
		gateway:    g,
		identifier: adapterNode.IEEEAddress,
		dd: &deviceDiscovery{
			gateway:        g,
			networkJoining: g.provider,
			eventSender:    g,
			logger:         g.logger,
		},
	}

	g.logger.LogInfo(g.ctx, "Adapter coordinator IEEE address.", logwrap.Datum("IEEEAddress", g.selfDevice.Identifier().String()))

	if err := g.provider.RegisterAdapterEndpoint(ctx, DefaultGatewayHomeAutomationEndpoint, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		g.logger.LogError(g.ctx, "Failed to register endpoint against adapter.", logwrap.Datum("Endpoint", DefaultGatewayHomeAutomationEndpoint), logwrap.Err(err))
		return err
	}

	go g.providerLoop()

	return nil
}

func (g *gateway) Stop(_ context.Context) error {
	g.logger.LogInfo(g.ctx, "Stopping ZDA.")
	g.selfDevice.dd.Stop()
	g.ctxCancel()
	return nil
}

var _ da.Gateway = (*gateway)(nil)

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
