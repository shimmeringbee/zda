package alarm_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/ias_zone"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	Alarms   map[capabilities.SensorType]bool
	Endpoint zigbee.Endpoint
	ZoneType uint16
}

type PersistentData struct {
	Alarms   map[string]bool
	Endpoint zigbee.Endpoint
	ZoneType uint16
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]Data
	datalock *sync.RWMutex
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.AlarmSensorFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.supervisor.ZCL().RegisterCommandLibrary(ias_zone.Register)

	i.supervisor.ZCL().Listen(func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
		return zclMessage.ClusterID == zcl.IASZoneId && zclMessage.CommandIdentifier == ias_zone.ZoneStatusChangeNotificationId
	}, i.zoneStatusChangeNotification)
}
