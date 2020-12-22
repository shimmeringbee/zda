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

		colorCapabilities := uint8(0)

		if results[color_control.ColorCapabilities].Status == 0 {
			colorCapabilities = uint8(results[color_control.ColorCapabilities].DataTypeValue.Value.(uint64))
		}

		data.SupportsHueSat = cfg.Bool("SupportsHueSat", colorCapabilities&0b00000001 > 0)
		data.SupportsXY = cfg.Bool("SupportsXY", colorCapabilities&0b00001000 > 0)
		data.SupportsTemperature = cfg.Bool("SupportsTemperature", colorCapabilities&0b00010000 > 0)

		hasCapability = data.SupportsXY || data.SupportsHueSat || data.SupportsTemperature

		if !hasCapability {
			i.supervisor.Logger().LogError(ctx, "Device has cluster, but ColorCapability attribute reports no suitable color spaces.", logwrap.Datum("Endpoint", data.Endpoint), logwrap.Datum("ColorCapabilities", colorCapabilities))
		}
	}

	if !hasCapability {
		i.attMonColorMode.Detach(ctx, d)
		i.attMonCurrentHue.Detach(ctx, d)
		i.attMonCurrentSat.Detach(ctx, d)
		i.attMonCurrentX.Detach(ctx, d)
		i.attMonCurrentY.Detach(ctx, d)
		i.attMonCurrentTemp.Detach(ctx, d)

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.ColorFlag)
	} else {
		i.supervisor.Logger().LogInfo(ctx, "Have Color capability.", logwrap.Datum("Endpoint", data.Endpoint), logwrap.Datum("SupportsHueSat", data.SupportsHueSat), logwrap.Datum("SupportsXY", data.SupportsXY), logwrap.Datum("SupportsTemperature", data.SupportsTemperature))

		if requiresPolling, err := i.attMonColorMode.Attach(ctx, d, data.Endpoint, nil); err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to attach Color Mode attribute monitor to device.", logwrap.Err(err))
			return err
		} else {
			i.supervisor.Logger().LogDebug(ctx, "Attached Color Mode attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
			data.RequiresPolling = requiresPolling
		}

		if data.SupportsXY {
			if requiresPolling, err := i.attMonCurrentX.Attach(ctx, d, data.Endpoint, nil); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach CurrentX attribute monitor to device.", logwrap.Err(err))
				return err
			} else {
				i.supervisor.Logger().LogDebug(ctx, "Attached CurrentX attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
				data.RequiresPolling = requiresPolling
			}

			if requiresPolling, err := i.attMonCurrentY.Attach(ctx, d, data.Endpoint, nil); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach CurrentY attribute monitor to device.", logwrap.Err(err))
				return err
			} else {
				i.supervisor.Logger().LogDebug(ctx, "Attached CurrentY attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
				data.RequiresPolling = requiresPolling
			}
		}

		if data.SupportsHueSat {
			if requiresPolling, err := i.attMonCurrentHue.Attach(ctx, d, data.Endpoint, nil); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Current Hue attribute monitor to device.", logwrap.Err(err))
				return err
			} else {
				i.supervisor.Logger().LogDebug(ctx, "Attached Current Hue attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
				data.RequiresPolling = requiresPolling
			}

			if requiresPolling, err := i.attMonCurrentSat.Attach(ctx, d, data.Endpoint, nil); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Current Sat attribute monitor to device.", logwrap.Err(err))
				return err
			} else {
				i.supervisor.Logger().LogDebug(ctx, "Attached Current Sat attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
				data.RequiresPolling = requiresPolling
			}
		}

		if data.SupportsTemperature {
			if requiresPolling, err := i.attMonCurrentTemp.Attach(ctx, d, data.Endpoint, nil); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Current Temperature attribute monitor to device.", logwrap.Err(err))
				return err
			} else {
				i.supervisor.Logger().LogDebug(ctx, "Attached Current Temperature attribute monitor.", logwrap.Datum("RequiresPolling", requiresPolling))
				data.RequiresPolling = requiresPolling
			}
		}

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.ColorFlag)
	}

	return nil
}
