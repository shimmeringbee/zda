package pressure_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

func (i *Implementation) Reading(ctx context.Context, dad da.Device) ([]capabilities.PressureReading, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return []capabilities.PressureReading{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.PressureSensorFlag) {
		return []capabilities.PressureReading{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return []capabilities.PressureReading{{Value: i.data[d.Identifier].State}}, nil
}
