package alarm_warning_device

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/ias_warning_device"
	"github.com/shimmeringbee/zda"
	"math"
	"time"
)

var _ capabilities.AlarmWarningDevice = (*Implementation)(nil)

func (i *Implementation) Alarm(ctx context.Context, dad da.Device, alarmType capabilities.AlarmType, volume float64, visual bool, duration time.Duration) error {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.Lock()
	i.data[d.Identifier].AlarmType = alarmType
	i.data[d.Identifier].Volume = volume
	i.data[d.Identifier].Visual = visual
	i.data[d.Identifier].AlarmUntil = time.Now().Add(duration)
	i.datalock.Unlock()

	i.pollWarningDevice(ctx, d)

	return nil
}

func (i *Implementation) Clear(ctx context.Context, dad da.Device) error {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.Lock()
	endpoint := i.data[d.Identifier].Endpoint
	i.data[d.Identifier].AlarmUntil = time.Time{}
	i.datalock.Unlock()

	return i.supervisor.ZCL().SendCommand(ctx, d, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
		WarningMode:     ias_warning_device.Stop,
		StrobeMode:      ias_warning_device.NoStrobe,
		WarningDuration: 0,
	})
}

func (i *Implementation) Alert(ctx context.Context, dad da.Device, alarmType capabilities.AlarmType, alertType capabilities.AlertType, volume float64, visual bool) error {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	endpoint := i.data[d.Identifier].Endpoint
	i.datalock.RUnlock()

	return i.supervisor.ZCL().SendCommand(ctx, d, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.Squawk{
		SquawkMode:  mapAlertTypeToSquawk(alertType),
		Strobe:      visual,
		SquawkLevel: mapVolumeToSquawkLevel(volume),
	})
}

func mapVolumeToSquawkLevel(volume float64) ias_warning_device.SquawkLevel {
	if volume <= 0.25 {
		return ias_warning_device.Low
	} else if volume <= 0.5 {
		return ias_warning_device.Medium
	} else if volume <= 0.75 {
		return ias_warning_device.High
	} else {
		return ias_warning_device.VeryHigh
	}
}

func mapAlertTypeToSquawk(alertType capabilities.AlertType) ias_warning_device.SquawkMode {
	switch alertType {
	case capabilities.DisarmAlert:
		return ias_warning_device.SystemDisarmed
	default:
		return ias_warning_device.SystemArmed
	}
}

func mapAlarmTypeToWarningMode(alarmType capabilities.AlarmType) ias_warning_device.WarningMode {
	switch alarmType {
	case capabilities.FireAlarm:
		return ias_warning_device.Fire
	case capabilities.SecurityAlarm:
		return ias_warning_device.Burglar
	default:
		return ias_warning_device.Emergency
	}
}

func (i *Implementation) Status(ctx context.Context, dad da.Device) (capabilities.WarningDeviceState, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return capabilities.WarningDeviceState{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return capabilities.WarningDeviceState{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	data := i.data[d.Identifier]

	remaining := time.Until(data.AlarmUntil)

	if data.AlarmUntil.IsZero() || remaining < 0 {
		return capabilities.WarningDeviceState{Warning: false}, nil
	}

	return capabilities.WarningDeviceState{
		Warning:           true,
		AlarmType:         data.AlarmType,
		Visual:            data.Visual,
		Volume:            data.Volume,
		DurationRemaining: remaining,
	}, nil
}

const MaximumDuration = 20 * time.Minute
const StopDuration = 5

func (i *Implementation) pollWarningDevice(ctx context.Context, d zda.Device) bool {
	i.datalock.RLock()
	data := *i.data[d.Identifier]
	i.datalock.RUnlock()

	if data.AlarmUntil.IsZero() {
		return true
	}

	remainingDuration := time.Until(data.AlarmUntil)
	limitedRemainingDuration := math.Ceil(float64(time.Duration(math.Min(float64(remainingDuration), float64(MaximumDuration))) / time.Second))

	strobeMode := ias_warning_device.NoStrobe
	warningMode := ias_warning_device.Stop

	if limitedRemainingDuration > 0 {
		if data.Visual {
			strobeMode = ias_warning_device.StrobeWithWarning
		}

		warningMode = mapAlarmTypeToWarningMode(data.AlarmType)
	} else if limitedRemainingDuration < 0 {
		if limitedRemainingDuration < StopDuration {
			i.datalock.Lock()
			i.data[d.Identifier].AlarmUntil = time.Time{}
			i.datalock.Unlock()
		}

		limitedRemainingDuration = 0
	}

	i.supervisor.Logger().LogDebug(ctx, "Sending periodic StartWarning message to Warning Device.", logwrap.Datum("Identifier", d.Identifier.String()), logwrap.Datum("RemainingDuration", limitedRemainingDuration), logwrap.Datum("WarningMode", warningMode), logwrap.Datum("StrobeMode", strobeMode))

	if err := i.supervisor.ZCL().SendCommand(ctx, d, data.Endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
		WarningMode:     warningMode,
		StrobeMode:      strobeMode,
		WarningDuration: uint16(limitedRemainingDuration),
	}); err != nil {
		i.supervisor.Logger().LogError(ctx, "Failed to send StartWarning message to Warning Device.", logwrap.Err(err), logwrap.Datum("Identifier", d.Identifier.String()))
	}

	return true
}
