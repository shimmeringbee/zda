package pressure_sensor

import (
	"context"
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
		State:                   i.data[d.Identifier].State,
		RequiresPolling:         i.data[d.Identifier].RequiresPolling,
		Endpoint:                i.data[d.Identifier].Endpoint,
		LastUpdateTime:          i.data[d.Identifier].LastUpdateTime,
		LastChangeTime:          i.data[d.Identifier].LastChangeTime,
		VendorXiaomiApproachOne: i.data[d.Identifier].VendorXiaomiApproachOne,
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

	i.attMonPressureMeasurementCluster.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	i.attMonVendorXiaomiApproachOne.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)

	i.data[d.Identifier] = Data{
		State:                   pd.State,
		RequiresPolling:         pd.RequiresPolling,
		Endpoint:                pd.Endpoint,
		LastUpdateTime:          pd.LastUpdateTime,
		LastChangeTime:          pd.LastChangeTime,
		VendorXiaomiApproachOne: pd.VendorXiaomiApproachOne,
	}

	return nil
}
