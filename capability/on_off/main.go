package on_off

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	State           bool
	RequiresPolling bool
	PollerCancel    func()
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
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.OnOffFlag
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.supervisor.ZCL().Listen(func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
		_, canCast := zclMessage.Command.(*global.ReportAttributes)
		return zclMessage.ClusterID == zcl.OnOffId && canCast
	}, i.zclCallback)

	i.supervisor.ZCL().RegisterCommandLibrary(onoff.Register)
}

func (i *Implementation) pollDevice(ctx context.Context, d zda.Device) bool {
	i.datalock.RLock()
	data, found := i.data[d.Identifier]
	i.datalock.RUnlock()

	if !found {
		return false
	}

	endpoint := data.Endpoint

	results, err := i.supervisor.ZCL().ReadAttributes(ctx, d, endpoint, zcl.OnOffId, []zcl.AttributeID{onoff.OnOff})
	if err == nil {
		if results[onoff.OnOff].Status == 0 {
			i.setState(d, results[onoff.OnOff].DataTypeValue.Value.(bool))
		}
	}

	return true
}
