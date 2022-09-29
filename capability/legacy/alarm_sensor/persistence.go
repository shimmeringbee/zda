package alarm_sensor

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
)

func (i *Implementation) DataStruct() interface{} {
	return &PersistentData{}
}

func (i *Implementation) Save(d zda.Device) (interface{}, error) {
	if !d.HasCapability(capabilities.AlarmSensorFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	savedAlarms := map[string]bool{}

	for k, v := range i.data[d.Identifier].Alarms {
		savedAlarms[k.String()] = v
	}

	return &PersistentData{
		Alarms:         savedAlarms,
		Endpoint:       i.data[d.Identifier].Endpoint,
		LastUpdateTime: i.data[d.Identifier].LastUpdateTime,
		LastChangeTime: i.data[d.Identifier].LastChangeTime,
		ZoneType:       i.data[d.Identifier].ZoneType,
	}, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.AlarmSensorFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*PersistentData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	restoredAlarms := map[capabilities.SensorType]bool{}

	for k, v := range pd.Alarms {
		for fK, fV := range capabilities.NameMapping {
			if k == fV {
				restoredAlarms[fK] = v
			}
		}
	}

	i.data[d.Identifier] = Data{
		Alarms:         restoredAlarms,
		Endpoint:       pd.Endpoint,
		LastUpdateTime: pd.LastUpdateTime,
		LastChangeTime: pd.LastChangeTime,
		ZoneType:       pd.ZoneType,
	}

	return nil
}
