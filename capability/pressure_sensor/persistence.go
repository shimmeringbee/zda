package pressure_sensor

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
	if !d.HasCapability(capabilities.PressureSensorFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return &PersistentData{
		State:           i.data[d.Identifier].State,
		RequiresPolling: i.data[d.Identifier].RequiresPolling,
		Endpoint:        i.data[d.Identifier].Endpoint,
	}, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.PressureSensorFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*PersistentData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	if i.data[d.Identifier].PollerCancel != nil {
		i.data[d.Identifier].PollerCancel()
	}

	var pollerCancelFn func()

	if pd.RequiresPolling {
		cfg := i.supervisor.DeviceConfig().Get(d, capabilities.StandardNames[capabilities.PressureSensorFlag])
		pollerCancelFn = i.supervisor.Poller().Add(d, cfg.Duration("PollingInterval", DefaultPollingInterval), i.pollDevice)
	}

	i.data[d.Identifier] = Data{
		State:           pd.State,
		RequiresPolling: pd.RequiresPolling,
		PollerCancel:    pollerCancelFn,
		Endpoint:        pd.Endpoint,
	}

	return nil
}
