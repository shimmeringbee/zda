package alarm_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

var _ capabilities.AlarmSensor = (*Implementation)(nil)

func (i *Implementation) Status(ctx context.Context, dad da.Device) (map[capabilities.SensorType]bool, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return nil, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.AlarmSensorFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	internalAlarms := i.data[d.Identifier].Alarms

	result := make(map[capabilities.SensorType]bool, len(internalAlarms))
	for k, v := range internalAlarms {
		result[k] = v
	}

	return result, nil
}
