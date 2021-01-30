package temperature_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
	"time"
)

type Data struct {
	State           float64
	RequiresPolling bool
	Endpoint        zigbee.Endpoint
	LastUpdateTime  time.Time
	LastChangeTime  time.Time
}

type PersistentData struct {
	State           float64
	RequiresPolling bool
	Endpoint        zigbee.Endpoint
	LastUpdateTime  time.Time
	LastChangeTime  time.Time
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

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.attributeMonitor = i.supervisor.AttributeMonitorCreator().Create(i, zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, i.attributeUpdate)
}

func (i *Implementation) attributeUpdate(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeSignedInt16 {
		value, ok := v.Value.(int64)

		if ok {
			i.setState(d, value)
		}
	}
}
