package color

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	da_color "github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type State struct {
	CurrentMode uint8

	CurrentX float64
	CurrentY float64

	CurrentHue float64
	CurrentSat float64

	CurrentTemperature float64
}

type Data struct {
	State           State
	RequiresPolling bool
	Endpoint        zigbee.Endpoint

	SupportsXY          bool
	SupportsHueSat      bool
	SupportsTemperature bool
}

type PersistentState struct {
	CurrentMode uint8

	CurrentX float64
	CurrentY float64

	CurrentHue float64
	CurrentSat float64

	CurrentTemperature float64
}

type PersistentData struct {
	State           PersistentState
	RequiresPolling bool
	Endpoint        zigbee.Endpoint

	SupportsXY          bool
	SupportsHueSat      bool
	SupportsTemperature bool
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]Data
	datalock *sync.RWMutex

	attMonColorMode   zda.AttributeMonitor
	attMonCurrentX    zda.AttributeMonitor
	attMonCurrentY    zda.AttributeMonitor
	attMonCurrentHue  zda.AttributeMonitor
	attMonCurrentSat  zda.AttributeMonitor
	attMonCurrentTemp zda.AttributeMonitor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.ColorFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.supervisor.ZCL().RegisterCommandLibrary(color_control.Register)

	i.attMonColorMode = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.ColorMode, zcl.TypeUnsignedInt8, i.attributeUpdate)

	i.attMonCurrentX = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentX, zcl.TypeUnsignedInt16, i.attributeUpdate)
	i.attMonCurrentY = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentY, zcl.TypeUnsignedInt16, i.attributeUpdate)

	i.attMonCurrentHue = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentHue, zcl.TypeUnsignedInt8, i.attributeUpdate)
	i.attMonCurrentSat = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentSaturation, zcl.TypeUnsignedInt8, i.attributeUpdate)

	i.attMonCurrentTemp = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.ColorTemperatureMireds, zcl.TypeUnsignedInt16, i.attributeUpdate)
}

func (i *Implementation) attributeUpdate(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	oldData := i.data[d.Identifier]
	newData := i.data[d.Identifier]

	if v.DataType == zcl.TypeEnum8 {
		data := v.Value.(uint8)

		switch a {
		case color_control.ColorMode:
			newData.State.CurrentMode = data
		}
	}

	if v.DataType == zcl.TypeUnsignedInt8 {
		data := uint8(v.Value.(uint64))

		switch a {
		case color_control.CurrentHue:
			newData.State.CurrentHue = float64(data) * 360.0 / 254.0
		case color_control.CurrentSaturation:
			newData.State.CurrentSat = float64(data) / 254.0
		}
	}

	if v.DataType == zcl.TypeUnsignedInt16 {
		data := uint16(v.Value.(uint64))

		switch a {
		case color_control.CurrentX:
			newData.State.CurrentX = float64(data) / 65536.0
		case color_control.CurrentY:
			newData.State.CurrentY = float64(data) / 65536.0
		case color_control.ColorTemperatureMireds:
			newData.State.CurrentTemperature = 1000000 / float64(data)
		}
	}

	if oldData.State != newData.State {
		i.data[d.Identifier] = newData

		i.supervisor.Logger().LogDebug(context.Background(), "Color state update received.", logwrap.Datum("Identifier", d.Identifier.String()), logwrap.Datum("State", newData.State))

		i.supervisor.DAEventSender().Send(capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  i.stateToColorStatus(newData.State),
		})
	}
}

func (i *Implementation) stateToColorStatus(state State) capabilities.ColorStatus {
	var mode capabilities.Mode
	var color da_color.ConvertibleColor
	var temperature float64

	switch state.CurrentMode {
	case 0x00:
		mode = capabilities.ColorMode
		color = da_color.HSVColor{
			Hue:   state.CurrentHue,
			Sat:   state.CurrentSat,
			Value: 1.0,
		}
	case 0x01:
		mode = capabilities.ColorMode
		color = da_color.XYColor{
			X:  state.CurrentX,
			Y:  state.CurrentY,
			Y2: 100.0,
		}
	case 0x02:
		mode = capabilities.TemperatureMode
		temperature = state.CurrentTemperature
	}

	return capabilities.ColorStatus{
		Mode: mode,
		Color: capabilities.ColorSettings{
			Current: color,
		},
		Temperature: capabilities.TemperatureSettings{
			Current: temperature,
		},
	}
}
