package on_off

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"time"
)

const DefaultPollingInterval = 5 * time.Second

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
	cfg := i.supervisor.DeviceConfig().Get(d, capabilities.StandardNames[capabilities.OnOffFlag])

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.OnOffId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	if !hasCapability {
		i.datalock.Lock()
		if i.data[d.Identifier].PollerCancel != nil {
			i.data[d.Identifier].PollerCancel()
		}

		i.data[d.Identifier] = OnOffData{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.OnOffFlag)
	} else {
		endpoint := zigbee.Endpoint(cfg.Int("Endpoint", int(endpoints[0])))

		var onOffData OnOffData
		onOffData.Endpoint = endpoint

		onOffData.RequiresPolling = cfg.Bool("RequiresPolling", onOffData.RequiresPolling)

		if !onOffData.RequiresPolling {
			err := i.supervisor.ZCL().Bind(ctx, d, onOffData.Endpoint, zcl.OnOffId)
			if err != nil {
				onOffData.RequiresPolling = true
			}

			minimumReportingInterval := cfg.Int("MinimumReportingInterval", 0)
			maximumReportingInterval := cfg.Int("MaximumReportingInterval", 60)

			err = i.supervisor.ZCL().ConfigureReporting(ctx, d, onOffData.Endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, uint16(minimumReportingInterval), uint16(maximumReportingInterval), nil)
			if err != nil {
				onOffData.RequiresPolling = true
			}
		}

		if onOffData.RequiresPolling {
			onOffData.PollerCancel = i.supervisor.Poller().Add(d, cfg.Duration("PollingInterval", DefaultPollingInterval), i.pollDevice)
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
