package color

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
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

	//i.attributeMonitor.Detach(ctx, d)
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

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.ColorControlId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	var data Data

	if hasCapability {
		data.Endpoint = zigbee.Endpoint(cfg.Int("Endpoint", int(selectEndpoint(endpoints, d.Endpoints))))

		results, err := i.supervisor.ZCL().ReadAttributes(ctx, d, data.Endpoint, zcl.ColorControlId, []zcl.AttributeID{color_control.ColorCapabilities})
		if err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to read color capabilities from color control cluster.", logwrap.Err(err))
		}

		data.SupportsHueSat = cfg.Bool("SupportsHueSat", results[color_control.ColorCapabilities].Status == 0 && results[color_control.ColorCapabilities].DataTypeValue.Value.(uint64)&0b00000001 > 0)
		data.SupportsXY = cfg.Bool("SupportsXY", results[color_control.ColorCapabilities].Status == 0 && results[color_control.ColorCapabilities].DataTypeValue.Value.(uint64)&0b00001000 > 0)
		data.SupportsTemperature = cfg.Bool("SupportsTemperature", results[color_control.ColorCapabilities].Status == 0 && results[color_control.ColorCapabilities].DataTypeValue.Value.(uint64)&0b00010000 > 0)
	}

	if !hasCapability {
		//i.attributeMonitor.Detach(ctx, d)

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.ColorFlag)
	} else {
		i.supervisor.Logger().LogInfo(ctx, "Have Color capability.", logwrap.Datum("Endpoint", data.Endpoint), logwrap.Datum("SupportsHueSat", data.SupportsHueSat), logwrap.Datum("SupportsXY", data.SupportsXY), logwrap.Datum("SupportsTemperature", data.SupportsTemperature))

		if data.SupportsXY {

		}

		if data.SupportsHueSat {

		}

		if data.SupportsTemperature {

		}

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.ColorFlag)
	}

	return nil
}
