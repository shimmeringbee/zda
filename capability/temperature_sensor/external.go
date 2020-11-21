package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

var _ capabilities.TemperatureSensor = (*Implementation)(nil)

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
