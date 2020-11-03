package power_supply

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
	if !d.HasCapability(capabilities.PowerSupplyFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return &PersistentData{
		PowerStatus: i.data[d.Identifier].PowerStatus,
	}, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.PowerSupplyFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*PersistentData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	i.data[d.Identifier] = Data{
		PowerStatus: pd.PowerStatus,
	}

	return nil
}
