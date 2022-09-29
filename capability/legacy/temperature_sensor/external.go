package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"time"
)

var _ capabilities.TemperatureSensor = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)

func (i *Implementation) Reading(ctx context.Context, dad da.Device) ([]capabilities.TemperatureReading, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return []capabilities.TemperatureReading{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.TemperatureSensorFlag) {
		return []capabilities.TemperatureReading{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return []capabilities.TemperatureReading{{Value: i.data[d.Identifier].State}}, nil
}

func (i *Implementation) LastChangeTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.TemperatureSensorFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastChangeTime, nil
}

func (i *Implementation) LastUpdateTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.TemperatureSensorFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastUpdateTime, nil
}
