package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zigbee"
	"time"
)

const (
	EnumerationDurationMax    = 1 * time.Minute
	EnumerationNetworkTimeout = 2 * time.Second
	EnumerationNetworkRetries = 5
)

type enumerateDevice struct {
	gw     *gateway
	logger logwrap.Logger

	nq zigbee.NodeQuerier
}

func (e enumerateDevice) onNodeJoin(ctx context.Context, join nodeJoin) error {
	//TODO implement me
	panic("implement me")
}

func (e enumerateDevice) Enumerate(ctx context.Context, d da.Device) error {
	if err := da.DeviceCapabilityCheck(d, e.gw, capabilities.EnumerateDeviceFlag); err != nil {
		return err
	}

	//TODO implement me
	panic("implement me")
}

func (e enumerateDevice) Status(ctx context.Context, d da.Device) (capabilities.EnumerationStatus, error) {
	if err := da.DeviceCapabilityCheck(d, e.gw, capabilities.EnumerateDeviceFlag); err != nil {
		return capabilities.EnumerationStatus{}, err
	}

	//TODO implement me
	panic("implement me")
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

	_, _ = e.interrogateNode(ctx, n)

}

func (e enumerateDevice) interrogateNode(ctx context.Context, n *node) (inventory, error) {
	var inv inventory
	inv.endpointDesc = make(map[zigbee.Endpoint]endpointDescription)

	e.logger.LogTrace(ctx, "Enumerating node description.")
	if nd, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) (zigbee.NodeDescription, error) {
		return e.nq.QueryNodeDescription(ctx, n.address)
	}); err != nil {
		e.logger.LogError(ctx, "Failed to enumerate node description.", logwrap.Err(err))
		return inventory{}, err
	} else {
		inv.desc = &nd
	}

	e.logger.LogTrace(ctx, "Enumerating node endpoints.")
	if eps, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) ([]zigbee.Endpoint, error) {
		return e.nq.QueryNodeEndpoints(ctx, n.address)
	}); err != nil {
		e.logger.LogError(ctx, "Failed to enumerate node endpoints.", logwrap.Err(err))
		return inventory{}, err
	} else {
		inv.endpoints = eps
	}

	for _, ep := range inv.endpoints {
		e.logger.LogTrace(ctx, "Enumerating node endpoint description.", logwrap.Datum("Endpoint", ep))
		if ed, err := retry.RetryWithValue(ctx, EnumerationNetworkTimeout, EnumerationNetworkRetries, func(ctx context.Context) (zigbee.EndpointDescription, error) {
			return e.nq.QueryNodeEndpointDescription(ctx, n.address, ep)
		}); err != nil {
			e.logger.LogError(ctx, "Failed to enumerate node endpoint description.", logwrap.Datum("Endpoint", ep), logwrap.Err(err))
			return inventory{}, err
		} else {
			inv.endpointDesc[ep] = endpointDescription{
				endpointDescription: ed,
			}
		}
	}

	return inv, nil
}

var _ capabilities.EnumerateDevice = (*enumerateDevice)(nil)
