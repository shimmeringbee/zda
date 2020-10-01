package on_off

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda"
	"time"
)

const PollInterval = 5 * time.Second

func (i *Implementation) AddedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[d.Identifier]; !found {
		i.data[d.Identifier] = OnOffData{}
	}

	return nil
}

func (i *Implementation) RemovedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if i.data[d.Identifier].PollerCancel != nil {
		i.data[d.Identifier].PollerCancel()
	}

	delete(i.data, d.Identifier)

	return nil
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	endpoints := zda.FindEndpointsWithClusterID(d, zcl.OnOffId)

	if len(endpoints) == 0 {
		i.datalock.Lock()
		if i.data[d.Identifier].PollerCancel != nil {
			i.data[d.Identifier].PollerCancel()
		}

		i.data[d.Identifier] = OnOffData{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.OnOffFlag)
	} else {
		endpoint := endpoints[0]

		var onOffData OnOffData
		onOffData.Endpoint = endpoint

		err := i.supervisor.ZCL().Bind(ctx, d, onOffData.Endpoint, zcl.OnOffId)
		if err != nil {
			onOffData.RequiresPolling = true
		}

		err = i.supervisor.ZCL().ConfigureReporting(ctx, d, onOffData.Endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, 0, 60, nil)
		if err != nil {
			onOffData.RequiresPolling = true
		}

		if onOffData.RequiresPolling {
			onOffData.PollerCancel = i.supervisor.Poller().Add(d, PollInterval, i.pollDevice)
		}

		i.datalock.Lock()
		i.data[d.Identifier] = onOffData
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.OnOffFlag)
	}

	return nil
}

func (i *Implementation) zclCallback(d zda.Device, m zcl.Message) {
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

func (i *Implementation) setState(d zda.Device, s bool) {
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
