package zda

import (
	"context"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"log"
	"time"
)

type ZigbeeDeviceDiscovery struct {
	gateway        Gateway
	networkJoining zigbee.NetworkJoining
	eventSender    eventSender

	discovering    bool
	allowTimer     *time.Timer
	allowExpiresAt time.Time
}

func (d *ZigbeeDeviceDiscovery) Capability() Capability {
	return DeviceDiscoveryFlag
}

func (d *ZigbeeDeviceDiscovery) Enable(ctx context.Context, device Device, duration time.Duration) error {
	if DeviceIsNotGatewaySelf(d.gateway, device) {
		return DeviceIsNotGatewaySelfDeviceError
	}

	if err := d.networkJoining.PermitJoin(ctx, true); err != nil {
		return err
	}

	if d.allowTimer != nil {
		d.allowTimer.Stop()
	}

	d.allowExpiresAt = time.Now().Add(duration)
	d.allowTimer = time.AfterFunc(duration, func() {
		if err := d.Disable(ctx, device); err != nil {
			log.Printf("error while denying discovery after duration: %+v", err)
		}
	})

	d.discovering = true

	d.eventSender.sendEvent(DeviceDiscoveryEnabled{
		Gateway:  d.gateway,
		Duration: duration,
	})
	return nil
}

func (d *ZigbeeDeviceDiscovery) Disable(ctx context.Context, device Device) error {
	if DeviceIsNotGatewaySelf(d.gateway, device) {
		return DeviceIsNotGatewaySelfDeviceError
	}

	if err := d.networkJoining.DenyJoin(ctx); err != nil {
		return err
	}

	d.discovering = false
	d.allowTimer = nil

	d.eventSender.sendEvent(DeviceDiscoveryDisabled{
		Gateway: d.gateway,
	})
	return nil
}

func (d *ZigbeeDeviceDiscovery) Status(ctx context.Context, device Device) (DeviceDiscoveryStatus, error) {
	if DeviceIsNotGatewaySelf(d.gateway, device) {
		return DeviceDiscoveryStatus{}, DeviceIsNotGatewaySelfDeviceError
	}

	remainingDuration := d.allowExpiresAt.Sub(time.Now())
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	return DeviceDiscoveryStatus{Discovering: d.discovering, RemainingDuration: remainingDuration}, nil
}

func (d *ZigbeeDeviceDiscovery) Stop() {
	if d.allowTimer != nil {
		d.allowTimer.Stop()
	}
}
