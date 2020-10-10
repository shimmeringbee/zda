package relative_humidity_sensor

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/relative_humidity_measurement"
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

func selectEndpoint(found []zigbee.Endpoint, device map[zigbee.Endpoint]zigbee.EndpointDescription) zigbee.Endpoint {
	if len(found) > 0 {
		return found[0]
	}

	if len(device) > 0 {
		for endpoint, _ := range device {
			return endpoint
		}
	}

	return 0
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	cfg := i.supervisor.DeviceConfig().Get(d, i.KeyName())

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.RelativeHumidityMeasurementId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	if !hasCapability {
		i.datalock.Lock()
		if i.data[d.Identifier].PollerCancel != nil {
			i.data[d.Identifier].PollerCancel()
		}

		i.data[d.Identifier] = Data{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.RelativeHumiditySensorFlag)
	} else {
		var data Data
		data.Endpoint = zigbee.Endpoint(cfg.Int("Endpoint", int(selectEndpoint(endpoints, d.Endpoints))))
		data.RequiresPolling = cfg.Bool("RequiresPolling", data.RequiresPolling)

		attemptBinding := cfg.Bool("AttemptBinding", true)
		attemptReporting := cfg.Bool("AttemptReporting", true)

		if attemptBinding {
			err := i.supervisor.ZCL().Bind(ctx, d, data.Endpoint, zcl.RelativeHumidityMeasurementId)
			if err != nil {
				data.RequiresPolling = true
			}
		}

		if attemptReporting {
			minimumReportingInterval := cfg.Int("MinimumReportingInterval", 0)
			maximumReportingInterval := cfg.Int("MaximumReportingInterval", 60)
			reportableChange := cfg.Int("ReportableChange", 0)

			err := i.supervisor.ZCL().ConfigureReporting(ctx, d, data.Endpoint, zcl.RelativeHumidityMeasurementId, relative_humidity_measurement.MeasuredValue, zcl.TypeUnsignedInt16, uint16(minimumReportingInterval), uint16(maximumReportingInterval), int16(reportableChange))
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

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.RelativeHumiditySensorFlag)
	}

	return nil
}

func (i *Implementation) zclCallback(d zda.Device, m zcl.Message) {
	if !d.HasCapability(capabilities.RelativeHumiditySensorFlag) {
		return
	}

	report, ok := m.Command.(*global.ReportAttributes)
	if !ok {
		return
	}

	for _, record := range report.Records {
		if record.Identifier == relative_humidity_measurement.MeasuredValue {
			value, ok := record.DataTypeValue.Value.(uint64)
			if ok {
				i.setState(d, value)
				return
			}
		}
	}
}

func (i *Implementation) setState(d zda.Device, s uint64) {
	i.datalock.Lock()

	data := i.data[d.Identifier]

	reading := float64(s) / 10000.0

	if data.State != reading {
		data.State = reading
		i.data[d.Identifier] = data

		i.supervisor.DAEventSender().Send(capabilities.RelativeHumiditySensorState{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  []capabilities.RelativeHumidityReading{{Value: reading}},
		})
	}

	i.datalock.Unlock()
}
