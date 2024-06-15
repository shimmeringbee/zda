package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/factory"
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"runtime/debug"
	"slices"
	"sort"
	"sync"
	"time"
)

const (
	EnumerationDurationMax    = 1 * time.Minute
	EnumerationNetworkTimeout = 2 * time.Second
	EnumerationNetworkRetries = 5
)

type deviceManager interface {
	createNextDevice(*node) *device
	removeDevice(context.Context, IEEEAddressWithSubIdentifier) bool
	attachCapabilityToDevice(d *device, c implcaps.ZDACapability)
	detachCapabilityFromDevice(d *device, c implcaps.ZDACapability)
}

type enumerateDevice struct {
	gw     *gateway
	dm     deviceManager
	logger logwrap.Logger

	nq                zigbee.NodeQuerier
	zclReadFn         func(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error)
	runRulesFn        func(rules.Input) (rules.Output, error)
	capabilityFactory func(string, implcaps.ZDAInterface) implcaps.ZDACapability
	es                eventSender
}

func (e enumerateDevice) onNodeJoin(ctx context.Context, join nodeJoin) error {
	if err := e.startEnumeration(ctx, join.n); err != nil {
		e.logger.LogInfo(ctx, "Failed to start enumeration of node on join.", logwrap.Datum("IEEEAddress", join.n.address.String()), logwrap.Err(err))
	}

	return nil
}

func (e enumerateDevice) startEnumeration(ctx context.Context, n *node) error {
	e.logger.LogInfo(ctx, "Request to enumerate node received.", logwrap.Datum("IEEEAddress", n.address.String()))

	if !n.enumerationSem.TryAcquire(1) {
		return fmt.Errorf("enumeration already in progress")
	}

	go e.enumerate(ctx, n)

	return nil
}

func (e enumerateDevice) enumerate(pctx context.Context, n *node) {
	n.enumerationState = true

	n.m.RLock()
	for _, d := range n.device {
		e.es.sendEvent(capabilities.EnumerateDeviceStart{Device: d})
	}
	n.m.RUnlock()

	defer func() {
		n.enumerationState = false
		n.enumerationSem.Release(1)

		n.m.RLock()
		for _, d := range n.device {
			d.m.RLock()
			status, _ := d.eda.Status(pctx)
			d.m.RUnlock()

			e.es.sendEvent(capabilities.EnumerateDeviceStopped{Device: d, Status: status})
		}
		n.m.RUnlock()
	}()

	ctx, cancel := context.WithTimeout(pctx, EnumerationDurationMax)
	defer cancel()

	ctx, segmentEnd := e.logger.Segment(ctx, "Node enumeration.", logwrap.Datum("IEEEAddress", n.address.String()))
	defer segmentEnd()

	inv, err := e.interrogateNode(ctx, n)
	if err != nil {
		e.logger.LogError(ctx, "Failed to interrogate node.", logwrap.Err(err))
		return
	}

	e.logger.LogTrace(ctx, "Running rules against node.")
	inv, err = e.runRules(inv)
	if err != nil {
		e.logger.LogError(ctx, "Failed to run rules against node.", logwrap.Err(err))
		return
	}

	e.logger.LogTrace(ctx, "Grouping endpoints and devices.")
	inventoryDevices := e.groupInventoryDevices(inv)

	did := e.updateNodeTable(ctx, n, inventoryDevices)

	for _, id := range inventoryDevices {
		d := did[id.deviceId]
		errs := e.updateCapabilitiesOnDevice(ctx, d, id)

		d.eda.m.Lock()
		d.eda.results = errs
		d.eda.m.Unlock()
	}
}

func (e enumerateDevice) interrogateNode(ctx context.Context, n *node) (inventory, error) {
	inv := inventory{
		endpoints: make(map[zigbee.Endpoint]endpointDetails),
	}

	e.logger.LogTrace(ctx, "Enumerating node description.")
	if nd, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) (zigbee.NodeDescription, error) {
		return e.nq.QueryNodeDescription(ctx, n.address)
	}); err != nil {
		e.logger.LogError(ctx, "Failed to enumerate node description.", logwrap.Err(err))
		return inventory{}, err
	} else {
		inv.description = &nd
	}

	e.logger.LogTrace(ctx, "Enumerating node endpoints.")
	eps, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) ([]zigbee.Endpoint, error) {
		return e.nq.QueryNodeEndpoints(ctx, n.address)
	})

	if err != nil {
		e.logger.LogError(ctx, "Failed to enumerate node endpoints.", logwrap.Err(err))
		return inventory{}, err
	}

	for _, ep := range eps {
		e.logger.LogTrace(ctx, "Enumerating node endpoint description.", logwrap.Datum("Endpoint", ep))
		if ed, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) (zigbee.EndpointDescription, error) {
			return e.nq.QueryNodeEndpointDescription(ctx, n.address, ep)
		}); err != nil {
			e.logger.LogError(ctx, "Failed to enumerate node endpoint description.", logwrap.Datum("Endpoint", ep), logwrap.Err(err))
			return inventory{}, err
		} else {
			inv.endpoints[ep] = endpointDetails{
				description: ed,
			}
		}
	}

	for ep, desc := range inv.endpoints {
		if contains(desc.description.InClusterList, zcl.BasicId) {
			e.logger.LogTrace(ctx, "Querying vendor information from endpoint.", logwrap.Datum("Endpoint", ep))

			resp, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) ([]global.ReadAttributeResponseRecord, error) {
				return e.zclReadFn(ctx, n.address, false, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, ep, n.nextTransactionSequence(), []zcl.AttributeID{basic.ManufacturerName, basic.ModelIdentifier, basic.ManufacturerVersionDetails, basic.SerialNumber})
			})

			if err != nil {
				e.logger.LogWarn(ctx, "Failed to query vendor information from Basic cluster.", logwrap.Datum("Endpoint", ep), logwrap.Err(err))
				continue
			}

			for _, r := range resp {
				if r.Status != 0 {
					e.logger.LogInfo(ctx, "Device returned negative status to read attribute from Basic cluster.", logwrap.Datum("Endpoint", ep), logwrap.Datum("Attribute", r.Identifier), logwrap.Datum("Status", r.Status))
					continue
				}

				value := r.DataTypeValue.Value.(string)

				switch r.Identifier {
				case basic.ManufacturerName:
					desc.productInformation.manufacturer = value
				case basic.ModelIdentifier:
					desc.productInformation.product = value
				case basic.ManufacturerVersionDetails:
					desc.productInformation.version = value
				case basic.SerialNumber:
					desc.productInformation.serial = value
				}
			}

			inv.endpoints[ep] = desc

			e.logger.LogInfo(ctx, "Vendor information read from Basic cluster.", logwrap.Datum("Endpoint", ep), logwrap.Datum("ProductData", desc.productInformation))
		}
	}

	return inv, nil
}

func (e enumerateDevice) runRules(inv inventory) (inventory, error) {
	input := inv.toRulesInput()

	for id := range inv.endpoints {
		input.Self = int(id)

		if o, err := e.runRulesFn(input); err != nil {
			return inventory{}, err
		} else {
			ep := inv.endpoints[id]
			ep.rulesOutput = o
			inv.endpoints[id] = ep
		}
	}

	return inv, nil
}

type inventoryDevice struct {
	deviceId  int
	endpoints []endpointDetails
}

func (e enumerateDevice) groupInventoryDevices(inv inventory) []inventoryDevice {
	devices := map[int]*inventoryDevice{}
	var endpoints []int

	for eid, ep := range inv.endpoints {
		invDev := &inventoryDevice{deviceId: int(eid), endpoints: []endpointDetails{ep}}
		devices[int(eid)] = invDev
		endpoints = append(endpoints, int(eid))
	}

	sort.Ints(endpoints)

	var outDevices []inventoryDevice
	for _, ep := range endpoints {
		outDevices = append(outDevices, *devices[ep])
	}

	return outDevices
}

func (e enumerateDevice) updateNodeTable(ctx context.Context, n *node, inventoryDevices []inventoryDevice) map[int]*device {
	ctx, end := e.logger.Segment(ctx, "Updating node table.")
	defer end()

	deviceIdMapping := map[int]*device{}
	var unsetDevice []*device = nil

	/* Look for devices that exist but don't have a deviceId. */
	n.m.RLock()
	for _, d := range n.device {
		d.m.RLock()
		if !d.deviceIdSet {
			unsetDevice = append(unsetDevice, d)
		}
		d.m.RUnlock()
	}
	n.m.RUnlock()

	/* Find existing devices that match the deviceId. */
	n.m.RLock()
	for _, i := range inventoryDevices {
		for _, d := range n.device {
			d.m.RLock()
			devId := d.deviceId
			d.m.RUnlock()

			if devId == i.deviceId {
				deviceIdMapping[i.deviceId] = d
				break
			}
		}
	}
	n.m.RUnlock()

	/* Create new devices for those that are missing. */
	for _, i := range inventoryDevices {
		if _, found := deviceIdMapping[i.deviceId]; !found {
			var d *device

			if len(unsetDevice) > 0 {
				d = unsetDevice[0]
				unsetDevice = unsetDevice[1:]
			} else {
				d = e.dm.createNextDevice(n)
			}

			d.m.Lock()
			d.deviceId = i.deviceId
			d.deviceIdSet = true
			d.m.Unlock()
			deviceIdMapping[i.deviceId] = d
			e.logger.LogTrace(ctx, "Added new device.", logwrap.Datum("DeviceId", i.deviceId), logwrap.Datum("NewIdentifier", d.Identifier().String()))
		}
	}

	/* Aggregate devices that should no longer be present on node. */
	var devicesToRemove []*device

	n.m.RLock()
	for _, d := range n.device {
		d.m.RLock()
		if _, found := deviceIdMapping[d.deviceId]; !found {
			e.logger.LogTrace(ctx, "Removing old device.", logwrap.Datum("DeviceId", d.deviceId), logwrap.Datum("OldIdentifier", d.Identifier().String()))
			devicesToRemove = append(devicesToRemove, d)
		}
		d.m.RUnlock()
	}
	n.m.RUnlock()

	for _, d := range devicesToRemove {
		e.dm.removeDevice(ctx, d.address)
	}

	return deviceIdMapping
}

func (e enumerateDevice) updateCapabilitiesOnDevice(ctx context.Context, d *device, id inventoryDevice) map[da.Capability]*capabilities.EnumerationCapability {
	ctx, end := e.logger.Segment(ctx, "Enumerating capabilities", logwrap.Datum("Identifier", d.Identifier().String()))
	defer end()

	errs := map[da.Capability]*capabilities.EnumerationCapability{
		capabilities.EnumerateDeviceFlag: {Attached: true},
	}

	var activeCapabilities []da.Capability

	d.m.Lock()

	for _, ep := range id.endpoints {
		for capImplName, settings := range ep.rulesOutput.Capabilities {
			cF, found := factory.Mapping[capImplName]
			if !found {
				e.logger.LogWarn(ctx, "Could not find implementation for capability.", logwrap.Datum("CapabilityImplementation", capImplName))
				errs[capabilities.EnumerateDeviceFlag].Errors = append(errs[capabilities.EnumerateDeviceFlag].Errors, fmt.Errorf("could not find capability in rule output: %s", capImplName))
				continue
			}

			if _, found := errs[cF]; !found {
				errs[cF] = &capabilities.EnumerationCapability{Attached: false}
			}

			ectx, end := e.logger.Segment(ctx, "Enumerating capability.", logwrap.Datum("Endpoint", ep.description.Endpoint), logwrap.Datum("DeviceId", ep.description.DeviceID), logwrap.Datum("CapabilityImplementation", capImplName), logwrap.Datum("Capability", capabilities.StandardNames[cF]))
			attached, err := e.enumerateCapabilityOnDevice(ectx, d, capImplName, cF, activeCapabilities, settings)
			if err != nil {
				errs[cF].Errors = append(errs[cF].Errors, err...)
			}
			errs[cF].Attached = attached

			if attached {
				activeCapabilities = append(activeCapabilities, cF)
			}

			end()
		}
	}

	for cf, impl := range d.capabilities {
		if !slices.Contains(activeCapabilities, cf) {
			errs[cf] = &capabilities.EnumerationCapability{Attached: false}

			e.logger.LogInfo(ctx, "Removing redundant capability implementation.", logwrap.Datum("Capability", capabilities.StandardNames[cf]))
			if err := impl.Detach(ctx, implcaps.NoLongerEnumerated); err != nil {
				e.logger.LogWarn(ctx, "Failed to detach redundant capability.", logwrap.Datum("RedundantCapabilityImplementationName", impl.ImplName()), logwrap.Err(err))
				errs[cf].Errors = append(errs[cf].Errors, fmt.Errorf("failed to detach redundant capabiltiy: %w", err))
			}

			e.dm.detachCapabilityFromDevice(d, impl)
		}
	}

	d.m.Unlock()

	return errs
}

func (e enumerateDevice) enumerateCapabilityOnDevice(ctx context.Context, d *device, capImplName string, cF da.Capability, activeCapabilities []da.Capability, settings map[string]any) (bool, []error) {
	var errs []error

	if slices.Contains(activeCapabilities, cF) {
		e.logger.LogWarn(ctx, "Multiple capabilities of the same type present on endpoint.")
		return false, []error{fmt.Errorf("multiple implementations of same category, last attempted: %s", capImplName)}
	}

	c, found := d.capabilities[cF]
	if found && c.ImplName() != capImplName {
		found = false

		e.logger.LogInfo(ctx, "Removing redundant capability implementation.", logwrap.Datum("RedundantCapabilityImplementationName", c.ImplName()))

		if err := c.Detach(ctx, implcaps.NoLongerEnumerated); err != nil {
			e.logger.LogWarn(ctx, "Failed to detach redundant capability.", logwrap.Datum("RedundantCapabilityImplementationName", c.ImplName()), logwrap.Err(err))
			errs = append(errs, fmt.Errorf("failed to detach conflicting capabiltiy: %w", err))
		}

		e.dm.detachCapabilityFromDevice(d, c)
	}

	if !found {
		if c = e.capabilityFactory(capImplName, e.gw.zdaInterface); c == nil {
			e.logger.LogError(ctx, "Failed to find implementation of capability.")
			return false, []error{fmt.Errorf("failed to find concrete implementation: %s", capImplName)}
		}

		section := e.gw.sectionForDevice(d.address).Section("capability", capabilities.StandardNames[cF])
		section.Set("implementation", capImplName)

		c.Init(d, section.Section("data"))
	}

	e.logger.LogInfo(ctx, "Attaching capability implementation.")
	defer func() {
		if r := recover(); r != nil {
			e.logger.LogPanic(ctx, "Capability paniced during enumeration!", logwrap.Datum("Panic", r), logwrap.Datum("Trace", string(debug.Stack())))
		}
	}()
	attached, err := c.Enumerate(ctx, settings)
	if err != nil {
		e.logger.LogWarn(ctx, "Errored while attaching new capability.", logwrap.Err(err), logwrap.Datum("Attached", attached))
		errs = append(errs, fmt.Errorf("error while attaching: %s: %w", capImplName, err))
	}

	if !attached {
		e.logger.LogWarn(ctx, "Failed to attach capability implementation.")
		if err := c.Detach(ctx, implcaps.FailedAttach); err != nil {
			e.logger.LogWarn(ctx, "Failed to detach failed attaching capability.", logwrap.Err(err))
			errs = append(errs, fmt.Errorf("failed to detach failed attach on capabiltiy: %s: %w", capImplName, err))
		}

		e.dm.detachCapabilityFromDevice(d, c)
	} else {
		e.dm.attachCapabilityToDevice(d, c)
		e.logger.LogInfo(ctx, "Capability attached successfully.")
	}

	return attached, errs
}

type enumeratedDeviceAttachment struct {
	node   *node
	device *device
	ed     *enumerateDevice

	m       *sync.RWMutex
	results map[da.Capability]*capabilities.EnumerationCapability
}

func (e enumeratedDeviceAttachment) Capability() da.Capability {
	return capabilities.EnumerateDeviceFlag
}

func (e enumeratedDeviceAttachment) Name() string {
	return capabilities.StandardNames[capabilities.EnumerateDeviceFlag]
}

func (e enumeratedDeviceAttachment) Enumerate(ctx context.Context) error {
	return e.ed.startEnumeration(ctx, e.node)
}

func (e enumeratedDeviceAttachment) Status(_ context.Context) (capabilities.EnumerationStatus, error) {
	e.m.RLock()
	defer e.m.RUnlock()

	ret := capabilities.EnumerationStatus{
		Enumerating:      e.node.enumerationState,
		CapabilityStatus: map[da.Capability]capabilities.EnumerationCapability{},
	}

	for k, v := range e.results {
		ret.CapabilityStatus[k] = *v
	}

	return ret, nil
}

var _ capabilities.EnumerateDevice = (*enumeratedDeviceAttachment)(nil)
var _ da.BasicCapability = (*enumeratedDeviceAttachment)(nil)
