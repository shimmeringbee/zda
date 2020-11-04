package power_supply

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
	if !d.HasCapability(capabilities.PowerSupplyFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	var mainsPD []capabilities.PowerMainsStatus
	var batteryPD []capabilities.PowerBatteryStatus

	for _, mains := range i.data[d.Identifier].Mains {
		mainsPD = append(mainsPD, *mains)
	}

	for _, battery := range i.data[d.Identifier].Battery {
		batteryPD = append(batteryPD, *battery)
	}

	return &PersistentData{
		Mains:           mainsPD,
		Battery:         batteryPD,
		RequiresPolling: i.data[d.Identifier].RequiresPolling,
		Endpoint:        i.data[d.Identifier].Endpoint,
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

	var dataMains []*capabilities.PowerMainsStatus

	for _, mains := range pd.Mains {
		dataMains = append(dataMains, &mains)
	}

	var dataBattery []*capabilities.PowerBatteryStatus

	for _, battery := range pd.Battery {
		dataBattery = append(dataBattery, &battery)
	}

	i.data[d.Identifier] = Data{
		Mains:           dataMains,
		Battery:         dataBattery,
		RequiresPolling: pd.RequiresPolling,
		Endpoint:        pd.Endpoint,
	}

	i.attMonMainsVoltage.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	i.attMonMainsFrequency.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	i.attMonBatteryVoltage.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	i.attMonBatteryPercentageRemaining.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)

	return nil
}
