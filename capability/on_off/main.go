package on_off

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	State           bool
	RequiresPolling bool
	Endpoint        zigbee.Endpoint
}

type PersistentData struct {
	State           bool
	RequiresPolling bool
	Endpoint        zigbee.Endpoint
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]Data
	datalock *sync.RWMutex

	attributeMonitor zda.AttributeMonitor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.TemperatureSensorFlag
}

func (i *Implementation) KeyName() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.attributeMonitor = i.supervisor.AttributeMonitorCreator().Create(i, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, i.attributeUpdate)
}

func (i *Implementation) attributeUpdate(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeBoolean {
		value, ok := v.Value.(bool)

		if ok {
			i.setState(d, value)
		}
	}
}
