package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
	"time"
)

type deviceDiscovery struct {
	gateway        da.Gateway
	networkJoining zigbee.NetworkJoining
	eventSender    eventSender

	discovering    bool
	allowTimer     *time.Timer
	allowExpiresAt time.Time

	logger logwrap.Logger
}

func (d *deviceDiscovery) Capability() da.Capability {
	return capabilities.DeviceDiscoveryFlag
}

func (d *deviceDiscovery) Name() string {
	return capabilities.StandardNames[d.Capability()]
}

func (d *deviceDiscovery) Enable(ctx context.Context, duration time.Duration) error {
	d.logger.LogInfo(ctx, "Invoking PermitJoin on Zigbee provider.", logwrap.Datum("Duration", duration))
	if err := d.networkJoining.PermitJoin(ctx, true); err != nil {
		d.logger.LogError(ctx, "Failed to PermitJoin on Zigbee provider.", logwrap.Err(err))
		return err
	}

	if d.allowTimer != nil {
		d.allowTimer.Stop()
	}

	d.allowExpiresAt = time.Now().Add(duration)
	d.allowTimer = time.AfterFunc(duration, func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.Disable(cctx); err != nil {
			d.logger.LogError(cctx, "Automatic timed DenyJoin failed.", logwrap.Err(err))
		}
	})

	d.discovering = true

	d.eventSender.sendEvent(capabilities.DeviceDiscoveryEnabled{
		Gateway:  d.gateway,
		Duration: duration,
	})
	return nil
}

func (d *deviceDiscovery) Disable(ctx context.Context) error {
	d.logger.LogInfo(ctx, "Invoking DenyJoin on Zigbee provider.")
	if err := d.networkJoining.DenyJoin(ctx); err != nil {
		d.logger.LogError(ctx, "Failed to DenyJoin on Zigbee provider.", logwrap.Err(err))
		return err
	}

	d.discovering = false
	d.allowTimer = nil
	d.allowExpiresAt = time.Time{}

	d.eventSender.sendEvent(capabilities.DeviceDiscoveryDisabled{
		Gateway: d.gateway,
	})
	return nil
}

func (d *deviceDiscovery) Status(ctx context.Context) (capabilities.DeviceDiscoveryStatus, error) {
	remainingDuration := d.allowExpiresAt.Sub(time.Now())
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	return capabilities.DeviceDiscoveryStatus{Discovering: d.discovering, RemainingDuration: remainingDuration}, nil
}

func (d *deviceDiscovery) Stop() {
	if d.allowTimer != nil {
		d.allowTimer.Stop()
	}
}

var _ capabilities.DeviceDiscovery = (*deviceDiscovery)(nil)
var _ da.BasicCapability = (*deviceDiscovery)(nil)
