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
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"sort"
	"time"
)

const (
	EnumerationDurationMax    = 1 * time.Minute
	EnumerationNetworkTimeout = 2 * time.Second
	EnumerationNetworkRetries = 5
)

type deviceManager interface {
	createNextDevice(*node) *device
	removeDevice(IEEEAddressWithSubIdentifier) bool
}

type enumerateDevice struct {
	gw     *gateway
	dm     deviceManager
	logger logwrap.Logger

	nq         zigbee.NodeQuerier
	zclReadFn  func(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error)
	runRulesFn func(rules.Input) (rules.Output, error)
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
	defer n.enumerationSem.Release(1)

	ctx, cancel := context.WithTimeout(pctx, EnumerationDurationMax)
	defer cancel()

	ctx, segmentEnd := e.logger.Segment(ctx, "Node enumeration.", logwrap.Datum("IEEEAddress", n.address.String()))
	defer segmentEnd()

	inv, err := e.interrogateNode(ctx, n)
	if err != nil {
		e.logger.LogError(ctx, "Failed to interrogate node.", logwrap.Err(err))
		return
	}

	inv, err = e.runRules(inv)
	if err != nil {
		e.logger.LogError(ctx, "Failed to run rules against node.", logwrap.Err(err))
		return
	}

	inventoryDevices := e.groupInventoryDevices(inv)

	_ = e.updateNodeTable(n, inventoryDevices)

	// End of node and device enumeration, now capability enumeration based upon data.
	// * Add new capabilities.
	// * Update/refresh existing capabilities.
	// * Delete capabilities that are no longer present.
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
		if Contains(desc.description.InClusterList, zcl.BasicId) {
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
	deviceId  uint16
	endpoints []endpointDetails
}

func (e enumerateDevice) groupInventoryDevices(inv inventory) []inventoryDevice {
	devices := map[uint16]*inventoryDevice{}

	for _, ep := range inv.endpoints {
		invDev := devices[ep.description.DeviceID]
		if invDev == nil {
			invDev = &inventoryDevice{deviceId: ep.description.DeviceID}
			devices[ep.description.DeviceID] = invDev
		}

		invDev.endpoints = append(invDev.endpoints, ep)

		sort.Slice(invDev.endpoints, func(i, j int) bool {
			return invDev.endpoints[i].description.Endpoint < invDev.endpoints[j].description.Endpoint
		})
	}

	var outDevices []inventoryDevice
	for _, invDev := range devices {
		outDevices = append(outDevices, *invDev)
	}

	sort.Slice(outDevices, func(i, j int) bool {
		return outDevices[i].deviceId < outDevices[j].deviceId
	})

	return outDevices
}

func (e enumerateDevice) updateNodeTable(n *node, inventoryDevices []inventoryDevice) map[uint16]*device {
	deviceIdMapping := map[uint16]*device{}

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
			d := e.dm.createNextDevice(n)
			d.m.Lock()
			d.deviceId = i.deviceId
			d.capabilities[capabilities.EnumerateDeviceFlag] = &enumeratedDeviceAttachment{
				node:   n,
				device: d,
			}
			d.m.Unlock()
			deviceIdMapping[i.deviceId] = d
		}
	}

	/* Report devices that should no longer be present on node. */
	var devicesToRemove []IEEEAddressWithSubIdentifier

	n.m.RLock()
	for _, d := range n.device {
		d.m.RLock()
		if _, found := deviceIdMapping[d.deviceId]; !found {
			devicesToRemove = append(devicesToRemove, d.address)
		}
		d.m.RUnlock()
	}
	n.m.RUnlock()

	for _, d := range devicesToRemove {
		e.dm.removeDevice(d)
	}

	return deviceIdMapping
}

type enumeratedDeviceAttachment struct {
	node   *node
	device *device
}

func (e enumeratedDeviceAttachment) Capability() da.Capability {
	//TODO implement me
	panic("implement me")
}

func (e enumeratedDeviceAttachment) Name() string {
	//TODO implement me
	panic("implement me")
}

func (e enumeratedDeviceAttachment) Enumerate(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (e enumeratedDeviceAttachment) Status(ctx context.Context) (capabilities.EnumerationStatus, error) {
	//TODO implement me
	panic("implement me")
}

var _ capabilities.EnumerateDevice = (*enumeratedDeviceAttachment)(nil)
var _ da.BasicCapability = (*enumeratedDeviceAttachment)(nil)
