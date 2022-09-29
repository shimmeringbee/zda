package relative_humidity_sensor

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"time"
)

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

	i.attMonRelativeHumidityMeasurementCluster.Detach(ctx, d)
	i.attMonVendorXiaomiApproachOne.Detach(ctx, d)
	delete(i.data, d.Identifier)

	return nil
}

func selectEndpoint(found []zigbee.Endpoint, device map[zigbee.Endpoint]zigbee.EndpointDescription) zigbee.Endpoint {
	if len(found) > 0 {
		return found[0]
	}

	if len(device) > 0 {
		for endpoint := range device {
			return endpoint
		}
	}

	return 0
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	cfg := i.supervisor.DeviceConfig().Get(d, i.Name())

	humidityEndpoints := zda.FindEndpointsWithClusterID(d, zcl.RelativeHumidityMeasurementId)
	basicEndpoints := zda.FindEndpointsWithClusterID(d, zcl.BasicId)

	var data Data

	hasZCL := false

	if cfg.Bool("HasRelativeHumidityMeasurementZCLCluster", len(humidityEndpoints) > 0) {
		data.Endpoint = zigbee.Endpoint(cfg.Int("Endpoint", int(selectEndpoint(humidityEndpoints, d.Endpoints))))

		i.supervisor.Logger().LogInfo(ctx, "Have Relative Humidity Sensor capability.", logwrap.Datum("Endpoint", data.Endpoint))

		reportableChange := uint(cfg.Int("ReportableChange", 10))
		if requiresPolling, err := i.attMonRelativeHumidityMeasurementCluster.Attach(ctx, d, data.Endpoint, reportableChange); err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to attach attribute monitor to device.", logwrap.Err(err))
			return err
		} else {
			i.supervisor.Logger().LogDebug(ctx, "Attached attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
			data.RequiresPolling = requiresPolling
		}

		hasZCL = true
	}

	hasXiaomi := false

	if cfg.Bool("HasVendorXiaomiApproachOne", false) {
		data.Endpoint = zigbee.Endpoint(cfg.Int("BasicEndpoint", int(selectEndpoint(basicEndpoints, d.Endpoints))))

		i.supervisor.Logger().LogInfo(ctx, "Have Relative Humidity Sensor capability.", logwrap.Datum("Endpoint", data.Endpoint))

		if requiresPolling, err := i.attMonVendorXiaomiApproachOne.Attach(ctx, d, data.Endpoint, nil); err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to attach attribute monitor to device.", logwrap.Err(err))
			return err
		} else {
			i.supervisor.Logger().LogDebug(ctx, "Attached attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
			data.RequiresPolling = requiresPolling
		}

		data.VendorXiaomiApproachOne = true

		hasXiaomi = true
	}

	if hasZCL || hasXiaomi {
		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.RelativeHumiditySensorFlag)
	} else {
		i.attMonRelativeHumidityMeasurementCluster.Detach(ctx, d)
		i.attMonVendorXiaomiApproachOne.Detach(ctx, d)

		i.datalock.Lock()

		i.data[d.Identifier] = Data{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.RelativeHumiditySensorFlag)
	}

	return nil
}

func (i *Implementation) setState(d zda.Device, reading float64) {
	i.datalock.Lock()

	data := i.data[d.Identifier]

	currentTime := time.Now()
	data.LastUpdateTime = currentTime

	if data.State != reading {
		data.LastChangeTime = currentTime
		data.State = reading

		i.supervisor.Logger().LogDebug(context.Background(), "Relative Humidity Sensor state update received.", logwrap.Datum("Identifier", d.Identifier.String()), logwrap.Datum("State", data.State))

		i.supervisor.DAEventSender().Send(capabilities.RelativeHumiditySensorState{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  []capabilities.RelativeHumidityReading{{Value: reading}},
		})
	}

	i.data[d.Identifier] = data
	i.datalock.Unlock()
}
