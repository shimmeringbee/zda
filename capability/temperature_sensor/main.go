package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/proprietary/xiaomi"
	"github.com/shimmeringbee/zigbee"
	"sync"
	"time"
)

type Data struct {
	State                   float64
	RequiresPolling         bool
	Endpoint                zigbee.Endpoint
	LastUpdateTime          time.Time
	LastChangeTime          time.Time
	VendorXiaomiApproachOne bool
}

type PersistentData struct {
	State                   float64
	RequiresPolling         bool
	Endpoint                zigbee.Endpoint
	LastUpdateTime          time.Time
	LastChangeTime          time.Time
	VendorXiaomiApproachOne bool
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]Data
	datalock *sync.RWMutex

	attMonTemperatureMeasurementCluster zda.AttributeMonitor
	attMonVendorXiaomiApproachOne       zda.AttributeMonitor
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

	i.attMonTemperatureMeasurementCluster = i.supervisor.AttributeMonitorCreator().Create(i, zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, i.attributeUpdateTemperatureMeasurementCluster)
	i.attMonVendorXiaomiApproachOne = i.supervisor.AttributeMonitorCreator().Create(i, zcl.BasicId, zcl.AttributeID(0xff01), zcl.TypeStringCharacter8, i.attributeUpdateVendorXiaomiApproachOne)
}

func (i *Implementation) attributeUpdateTemperatureMeasurementCluster(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeSignedInt16 {
		value, ok := v.Value.(int64)

		if ok {
			i.setState(d, (float64(value)/100.0)+273.15)
		}
	}
}

func (i *Implementation) attributeUpdateVendorXiaomiApproachOne(d zda.Device, a zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeStringCharacter8 {
		value, ok := v.Value.(string)

		if ok {
			xal, err := xiaomi.ParseAttributeList([]byte(value))
			if err != nil {
				i.supervisor.Logger().LogError(context.Background(), "Failed to parse Xiaomi attribute list.", logwrap.Datum("Identifier", d.Identifier.String()), logwrap.Err(err))
				return
			}

			att, ok := xal[0x64]
			if ok && att.Attribute.DataType == zcl.TypeSignedInt16 {
				temp := float64(att.Attribute.Value.(int64))/100.0 + 273.15
				i.setState(d, temp)
			}
		}
	}
}
