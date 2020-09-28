package on_off

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda/capability"
	"time"
)

const PollInterval = 5 * time.Second

func (i *Implementation) addedDeviceCallback(ctx context.Context, e capability.AddedDevice) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[e.Device.Identifier]; !found {
		i.data[e.Device.Identifier] = OnOffData{}
	}

	return nil
}

func (i *Implementation) removedDeviceCallback(ctx context.Context, e capability.RemovedDevice) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if i.data[e.Device.Identifier].PollerCancel != nil {
		i.data[e.Device.Identifier].PollerCancel()
	}

	delete(i.data, e.Device.Identifier)

	return nil
}

func (i *Implementation) enumerateDeviceCallback(ctx context.Context, e capability.EnumerateDevice) error {
	endpoints := capability.FindEndpointsWithClusterID(e.Device, zcl.OnOffId)

	if len(endpoints) == 0 {
		i.datalock.Lock()
		if i.data[e.Device.Identifier].PollerCancel != nil {
			i.data[e.Device.Identifier].PollerCancel()
		}

		i.data[e.Device.Identifier] = OnOffData{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(e.Device, capabilities.OnOffFlag)
	} else {
		endpoint := endpoints[0]

		var onOffData OnOffData
		onOffData.Endpoint = endpoint

		err := i.supervisor.ZCL().Bind(ctx, e.Device, onOffData.Endpoint, zcl.OnOffId)
		if err != nil {
			onOffData.RequiresPolling = true
		}

		err = i.supervisor.ZCL().ConfigureReporting(ctx, e.Device, onOffData.Endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, 0, 60, nil)
		if err != nil {
			onOffData.RequiresPolling = true
		}

		if onOffData.RequiresPolling {
			onOffData.PollerCancel = i.supervisor.Poller().Add(e.Device, PollInterval, i.pollDevice)
		}

		i.datalock.Lock()
		i.data[e.Device.Identifier] = onOffData
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(e.Device, capabilities.OnOffFlag)
	}

	return nil
}

func (i *Implementation) zclCallback(d capability.Device, m zcl.Message) {
	if !d.HasCapability(capabilities.OnOffFlag) {
		return
	}

	report, ok := m.Command.(*global.ReportAttributes)
	if !ok {
		return
	}

	for _, record := range report.Records {
		if record.Identifier == onoff.OnOff {
			value, ok := record.DataTypeValue.Value.(bool)
			if ok {
				i.setState(d, value)
				return
			}
		}
	}
}

func (i *Implementation) setState(d capability.Device, s bool) {
	i.datalock.Lock()

	data := i.data[d.Identifier]

	if data.State != s {
		data.State = s
		i.data[d.Identifier] = data

		i.supervisor.DAEventSender().Send(capabilities.OnOffState{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  s,
		})
	}

	i.datalock.Unlock()
}
