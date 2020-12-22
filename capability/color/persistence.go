package color

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
	if !d.HasCapability(capabilities.ColorFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	state := i.data[d.Identifier].State

	return &PersistentData{
		State: PersistentState{
			CurrentMode:        state.CurrentMode,
			CurrentX:           state.CurrentX,
			CurrentY:           state.CurrentY,
			CurrentHue:         state.CurrentHue,
			CurrentSat:         state.CurrentSat,
			CurrentTemperature: state.CurrentTemperature,
		},
		RequiresPolling:     i.data[d.Identifier].RequiresPolling,
		Endpoint:            i.data[d.Identifier].Endpoint,
		SupportsXY:          i.data[d.Identifier].SupportsXY,
		SupportsHueSat:      i.data[d.Identifier].SupportsHueSat,
		SupportsTemperature: i.data[d.Identifier].SupportsTemperature,
	}, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.ColorFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*PersistentData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	i.attMonColorMode.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)

	if pd.SupportsHueSat {
		i.attMonCurrentHue.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
		i.attMonCurrentSat.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	}

	if pd.SupportsXY {
		i.attMonCurrentX.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
		i.attMonCurrentY.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	}

	if pd.SupportsTemperature {
		i.attMonCurrentTemp.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)
	}

	i.data[d.Identifier] = Data{
		State: State{
			CurrentMode:        pd.State.CurrentMode,
			CurrentX:           pd.State.CurrentX,
			CurrentY:           pd.State.CurrentY,
			CurrentHue:         pd.State.CurrentHue,
			CurrentSat:         pd.State.CurrentSat,
			CurrentTemperature: pd.State.CurrentTemperature,
		},
		RequiresPolling:     pd.RequiresPolling,
		Endpoint:            pd.Endpoint,
		SupportsXY:          pd.SupportsXY,
		SupportsHueSat:      pd.SupportsHueSat,
		SupportsTemperature: pd.SupportsTemperature,
	}

	return nil
}
