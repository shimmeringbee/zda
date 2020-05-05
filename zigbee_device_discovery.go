package zda

import (
	"context"
	"errors"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"log"
	"time"
)

type ZigbeeDeviceDiscovery struct {
	gateway        *ZigbeeGateway
	discovering    bool
	allowTimer     *time.Timer
	allowExpiresAt time.Time
}

var DeviceIsNotSelfOfGateway = errors.New("can not operate on device which is not the gateway")

func deviceIsNotGatewaySelfDevice(gateway Gateway, device Device) bool {
	return device.Gateway != gateway || device.Identifier != gateway.Self().Identifier
}

func (d *ZigbeeDeviceDiscovery) Enable(ctx context.Context, device Device, duration time.Duration) error {
	if deviceIsNotGatewaySelfDevice(d.gateway, device) {
		return DeviceIsNotSelfOfGateway
	}

	if err := d.gateway.provider.PermitJoin(ctx, true); err != nil {
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

	d.gateway.sendEvent(DeviceDiscoveryAllowed{
		Gateway:  d.gateway,
		Duration: duration,
	})
	return nil
}

func (d *ZigbeeDeviceDiscovery) Disable(ctx context.Context, device Device) error {
	if deviceIsNotGatewaySelfDevice(d.gateway, device) {
		return DeviceIsNotSelfOfGateway
	}

	if err := d.gateway.provider.DenyJoin(ctx); err != nil {
		return err
	}

	d.discovering = false
	d.allowTimer = nil

	d.gateway.sendEvent(DeviceDiscoveryDenied{
		Gateway: d.gateway,
	})
	return nil
}

func (d *ZigbeeDeviceDiscovery) Status(ctx context.Context, device Device) (DeviceDiscoveryStatus, error) {
	if deviceIsNotGatewaySelfDevice(d.gateway, device) {
		return DeviceDiscoveryStatus{}, DeviceIsNotSelfOfGateway
	}

	remainingDuration := d.allowExpiresAt.Sub(time.Now())
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	return DeviceDiscoveryStatus{Discovering: d.discovering, RemainingDuration: remainingDuration}, nil
}
