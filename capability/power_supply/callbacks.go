package power_supply

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
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

	powerStatus := capabilities.PowerStatus{}
	hasCapability := false

	if cfg.Bool("CheckBasicPowerSource", true) {
		basicEndpoints := zda.FindEndpointsWithClusterID(d, zcl.BasicId)
		basicEndpoint := zigbee.Endpoint(cfg.Int("BasicEndpoint", int(selectEndpoint(basicEndpoints, d.Endpoints))))

		basicResp, err := i.supervisor.ZCL().ReadAttributes(ctx, d, basicEndpoint, zcl.BasicId, []zcl.AttributeID{basic.PowerSource})
		if err != nil {
			return err
		}

		if basicResp[basic.PowerSource].Status == 0 {
			value := basicResp[basic.PowerSource].DataTypeValue.Value.(uint8)

			if (value&0x80) == 0x80 || (value&0x7f) == 0x03 {
				powerStatus.Battery = append(powerStatus.Battery, capabilities.PowerBatteryStatus{
					Available: true,
					Present:   capabilities.Available,
				})
			}

			if (value & 0x7f) != 0x03 {
				powerStatus.Mains = append(powerStatus.Mains, capabilities.PowerMainsStatus{
					Available: true,
					Present:   capabilities.Available,
				})
			}

			hasCapability = true
		}
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[d.Identifier]

	if hasCapability {
		data.PowerStatus = powerStatus

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.PowerSupplyFlag)
	} else {
		data.PowerStatus = capabilities.PowerStatus{}

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.PowerSupplyFlag)
	}

	i.data[d.Identifier] = data

	return nil
}
