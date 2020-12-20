package level

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/level"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	State           float64
	RequiresPolling bool
	Endpoint        zigbee.Endpoint
}

type PersistentData struct {
	State           float64
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
	return capabilities.LevelFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.attributeMonitor = i.supervisor.AttributeMonitorCreator().Create(i, zcl.LevelControlId, level.CurrentLevel, zcl.TypeUnsignedInt8, i.attributeUpdate)

	i.supervisor.ZCL().RegisterCommandLibrary(level.Register)
}

func (i *Implementation) attributeUpdate(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeUnsignedInt8 {
		value, ok := v.Value.(uint64)

		if ok {
			i.setState(d, value)
		}
	}
}
