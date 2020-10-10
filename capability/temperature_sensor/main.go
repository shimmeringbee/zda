package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	State           float64
	RequiresPolling bool
	PollerCancel    func()
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
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.TemperatureSensorFlag
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.supervisor.ZCL().Listen(func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
		_, canCast := zclMessage.Command.(*global.ReportAttributes)
		return zclMessage.ClusterID == zcl.TemperatureMeasurementId && canCast
	}, i.zclCallback)
}

func (i *Implementation) pollDevice(ctx context.Context, d zda.Device) bool {
	i.datalock.RLock()
	data, found := i.data[d.Identifier]
	i.datalock.RUnlock()

	if !found {
		return false
	}

	endpoint := data.Endpoint

	results, err := i.supervisor.ZCL().ReadAttributes(ctx, d, endpoint, zcl.TemperatureMeasurementId, []zcl.AttributeID{temperature_measurement.MeasuredValue})
	if err == nil {
		if results[temperature_measurement.MeasuredValue].Status == 0 {
			i.setState(d, results[temperature_measurement.MeasuredValue].DataTypeValue.Value.(int64))
		}
	}

	return true
}
