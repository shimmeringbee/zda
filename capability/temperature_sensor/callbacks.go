package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"time"
)

const DefaultPollingInterval = 5 * time.Second

func (i *Implementation) AddedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[d.Identifier]; !found {
		i.data[d.Identifier] = Data{}
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
	cfg := i.supervisor.DeviceConfig().Get(d, capabilities.StandardNames[capabilities.TemperatureSensorFlag])

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.TemperatureMeasurementId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	if !hasCapability {
		i.datalock.Lock()
		if i.data[d.Identifier].PollerCancel != nil {
			i.data[d.Identifier].PollerCancel()
		}

		i.data[d.Identifier] = Data{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.TemperatureSensorFlag)
	} else {
		endpoint := zigbee.Endpoint(cfg.Int("Endpoint", int(endpoints[0])))

		var data Data
		data.Endpoint = endpoint

		data.RequiresPolling = cfg.Bool("RequiresPolling", data.RequiresPolling)

		if !data.RequiresPolling {
			err := i.supervisor.ZCL().Bind(ctx, d, data.Endpoint, zcl.TemperatureMeasurementId)
			if err != nil {
				data.RequiresPolling = true
			}

			minimumReportingInterval := cfg.Int("MinimumReportingInterval", 0)
			maximumReportingInterval := cfg.Int("MaximumReportingInterval", 60)
			reportableChange := cfg.Int("ReportableChange", 0)

			err = i.supervisor.ZCL().ConfigureReporting(ctx, d, data.Endpoint, zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, uint16(minimumReportingInterval), uint16(maximumReportingInterval), int16(reportableChange))
			if err != nil {
				data.RequiresPolling = true
			}
		}

		if data.RequiresPolling {
			data.PollerCancel = i.supervisor.Poller().Add(d, cfg.Duration("PollingInterval", DefaultPollingInterval), i.pollDevice)
		}

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.TemperatureSensorFlag)
	}

	return nil
}

func (i *Implementation) zclCallback(d zda.Device, m zcl.Message) {
	if !d.HasCapability(capabilities.TemperatureSensorFlag) {
		return
	}

	report, ok := m.Command.(*global.ReportAttributes)
	if !ok {
		return
	}

	for _, record := range report.Records {
		if record.Identifier == temperature_measurement.MeasuredValue {
			value, ok := record.DataTypeValue.Value.(int16)
			if ok {
				i.setState(d, value)
				return
			}
		}
	}
}

func (i *Implementation) setState(d zda.Device, s int16) {
	i.datalock.Lock()

	data := i.data[d.Identifier]

	reading := float64(s) / 100.0

	if data.State != reading {
		data.State = reading
		i.data[d.Identifier] = data

		i.supervisor.DAEventSender().Send(capabilities.TemperatureSensorState{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  []capabilities.TemperatureReading{{Value: reading}},
		})
	}

	i.datalock.Unlock()
}
