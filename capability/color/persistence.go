package color

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
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

	persistentColor := PersistentColor{ColorSpace: state.CurrentColor.NativeColorspace()}

	switch persistentColor.ColorSpace {
	case color.XYY:
		x, y, y2 := state.CurrentColor.XYY()
		persistentColor.X = x
		persistentColor.Y = y
		persistentColor.Y2 = y2
	case color.HueSat:
		h, s, v := state.CurrentColor.HSV()
		persistentColor.Hue = h
		persistentColor.Sat = s
		persistentColor.Value = v
	case color.SRGB:
		r, g, b := state.CurrentColor.RGB()
		persistentColor.R = r
		persistentColor.G = g
		persistentColor.B = b
	}

	return &PersistentData{
		State: PersistentState{
			CurrentMode:        state.CurrentMode,
			CurrentColor:       persistentColor,
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

	//i.attributeMonitor.Reattach(context.Background(), d, pd.Endpoint, pd.RequiresPolling)

	var concreteColor color.ConvertibleColor

	switch pd.State.CurrentColor.ColorSpace {
	case color.XYY:
		concreteColor = color.XYColor{
			X:  pd.State.CurrentColor.X,
			Y:  pd.State.CurrentColor.Y,
			Y2: pd.State.CurrentColor.Y2,
		}
	case color.HueSat:
		concreteColor = color.HSVColor{
			Hue:   pd.State.CurrentColor.Hue,
			Sat:   pd.State.CurrentColor.Sat,
			Value: pd.State.CurrentColor.Value,
		}
	case color.SRGB:
		concreteColor = color.SRGBColor{
			R: pd.State.CurrentColor.R,
			G: pd.State.CurrentColor.G,
			B: pd.State.CurrentColor.B,
		}
	}

	i.data[d.Identifier] = Data{
		State: State{
			CurrentMode:        pd.State.CurrentMode,
			CurrentColor:       concreteColor,
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
