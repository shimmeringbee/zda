package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zigbee"
	"time"
)

type ZigbeeDeviceDiscovery struct {
	gateway        da.Gateway
	networkJoining zigbee.NetworkJoining
	eventSender    eventSender

	discovering    bool
	allowTimer     *time.Timer
	allowExpiresAt time.Time

	logger logwrap.Logger
}

func (d *ZigbeeDeviceDiscovery) Capability() da.Capability {
	return capabilities.DeviceDiscoveryFlag
}

func (d *ZigbeeDeviceDiscovery) Name() string {
	return capabilities.StandardNames[d.Capability()]
}

func (d *ZigbeeDeviceDiscovery) Enable(ctx context.Context, device da.Device, duration time.Duration) error {
	if da.DeviceIsNotGatewaySelf(d.gateway, device) {
		return da.DeviceIsNotGatewaySelfDeviceError
	}

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
		if err := d.Disable(ctx, device); err != nil {
			d.logger.LogError(ctx, "Automatic timed DenyJoin failed.", logwrap.Err(err))
		}
	})

	d.discovering = true

	d.eventSender.sendEvent(capabilities.DeviceDiscoveryEnabled{
		Gateway:  d.gateway,
		Duration: duration,
	})
	return nil
}

func (d *ZigbeeDeviceDiscovery) Disable(ctx context.Context, device da.Device) error {
	if da.DeviceIsNotGatewaySelf(d.gateway, device) {
		return da.DeviceIsNotGatewaySelfDeviceError
	}

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

func (d *ZigbeeDeviceDiscovery) Status(ctx context.Context, device da.Device) (capabilities.DeviceDiscoveryStatus, error) {
	if da.DeviceIsNotGatewaySelf(d.gateway, device) {
		return capabilities.DeviceDiscoveryStatus{}, da.DeviceIsNotGatewaySelfDeviceError
	}

	remainingDuration := d.allowExpiresAt.Sub(time.Now())
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	return capabilities.DeviceDiscoveryStatus{Discovering: d.discovering, RemainingDuration: remainingDuration}, nil
}

func (d *ZigbeeDeviceDiscovery) Start() {
}

func (d *ZigbeeDeviceDiscovery) Stop() {
	if d.allowTimer != nil {
		d.allowTimer.Stop()
	}
}
