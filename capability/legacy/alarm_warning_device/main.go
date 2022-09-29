package alarm_warning_device

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl/commands/local/ias_warning_device"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
	"time"
)

type Data struct {
	Endpoint zigbee.Endpoint

	PollerCancel func()
	AlarmUntil   time.Time
	AlarmType    capabilities.AlarmType
	Volume       float64
	Visual       bool
}

type PersistentData struct {
	Endpoint zigbee.Endpoint
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]*Data
	datalock *sync.RWMutex
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.AlarmWarningDeviceFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{}
	i.datalock = &sync.RWMutex{}

	i.supervisor.ZCL().RegisterCommandLibrary(ias_warning_device.Register)
}
