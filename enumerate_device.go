package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"time"
)

const (
	EnumerationDurationMax    = 1 * time.Minute
	EnumerationNetworkTimeout = 3000 * time.Millisecond
	EnumerationNetworkRetries = 5
)

type enumerateDevice struct {
	gw     *gateway
	logger logwrap.Logger
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

}

var _ capabilities.EnumerateDevice = (*enumerateDevice)(nil)
