package relative_humidity_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

func (i *Implementation) Reading(ctx context.Context, dad da.Device) ([]capabilities.RelativeHumidityReading, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return []capabilities.RelativeHumidityReading{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.RelativeHumiditySensorFlag) {
		return []capabilities.RelativeHumidityReading{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return []capabilities.RelativeHumidityReading{{Value: i.data[d.Identifier].State}}, nil
}
