package alarm_warning_device

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
	if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return &PersistentData{
		Endpoint: i.data[d.Identifier].Endpoint,
	}, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.AlarmWarningDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*PersistentData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	i.data[d.Identifier] = &Data{
		Endpoint:     pd.Endpoint,
		PollerCancel: i.supervisor.Poller().Add(d, AnnouncementPeriod, i.pollWarningDevice),
	}

	return nil
}
