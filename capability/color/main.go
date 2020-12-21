package color

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type State struct {
	CurrentMode        capabilities.Mode
	CurrentColor       color.ConvertibleColor
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

type PersistentColor struct {
	ColorSpace color.NativeColorspace

	X  float64 `json:",omitempty"`
	Y  float64 `json:",omitempty"`
	Y2 float64 `json:",omitempty"`

	Hue   float64 `json:",omitempty"`
	Sat   float64 `json:",omitempty"`
	Value float64 `json:",omitempty"`

	R uint8 `json:",omitempty"`
	G uint8 `json:",omitempty"`
	B uint8 `json:",omitempty"`
}

type PersistentState struct {
	CurrentMode        capabilities.Mode
	CurrentColor       PersistentColor
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

	i.attMonColorMode = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.ColorMode, zcl.TypeUnsignedInt8, nil)

	i.attMonCurrentX = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentX, zcl.TypeUnsignedInt16, nil)
	i.attMonCurrentY = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentY, zcl.TypeUnsignedInt16, nil)

	i.attMonCurrentHue = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentHue, zcl.TypeUnsignedInt8, nil)
	i.attMonCurrentSat = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.CurrentSaturation, zcl.TypeUnsignedInt8, nil)

	i.attMonCurrentTemp = i.supervisor.AttributeMonitorCreator().Create(i, zcl.ColorControlId, color_control.ColorTemperatureMireds, zcl.TypeUnsignedInt16, nil)
}
