package power_supply

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
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

	mains := capabilities.PowerMainsStatus{}
	battery := capabilities.PowerBatteryStatus{}

	basicEndpoints := zda.FindEndpointsWithClusterID(d, zcl.BasicId)
	if cfg.Bool("HasBasicPowerSource", len(basicEndpoints) > 0) {
		basicEndpoint := zigbee.Endpoint(cfg.Int("BasicEndpoint", int(selectEndpoint(basicEndpoints, d.Endpoints))))

		basicResp, err := i.supervisor.ZCL().ReadAttributes(ctx, d, basicEndpoint, zcl.BasicId, []zcl.AttributeID{basic.PowerSource})
		if err != nil {
			return err
		}

		if basicResp[basic.PowerSource].Status == 0 {
			value := basicResp[basic.PowerSource].DataTypeValue.Value.(uint8)

			if (value&0x80) == 0x80 || (value&0x7f) == 0x03 {
				battery.Available = true
				battery.Present |= capabilities.Available
			}

			if (value & 0x7f) != 0x03 {
				mains.Available = true
				mains.Present |= capabilities.Available
			}
		}
	}

	pcEndpoints := zda.FindEndpointsWithClusterID(d, zcl.PowerConfigurationId)
	if cfg.Bool("HasPowerConfiguration", len(pcEndpoints) > 0) {
		pcEndpoint := zigbee.Endpoint(cfg.Int("PowerConfigurationEndpoint", int(selectEndpoint(pcEndpoints, d.Endpoints))))

		pcResp, err := i.supervisor.ZCL().ReadAttributes(ctx, d, pcEndpoint, zcl.PowerConfigurationId, []zcl.AttributeID{power_configuration.MainsVoltage, power_configuration.MainsFrequency, power_configuration.BatteryVoltage, power_configuration.BatteryPercentageRemaining, power_configuration.BatteryRatedVoltage})
		if err != nil {
			return err
		}

		if pcResp[power_configuration.MainsVoltage].Status == 0 {
			voltage := float64(pcResp[power_configuration.MainsVoltage].DataTypeValue.Value.(uint64)) / 10.0

			mains.Present |= capabilities.Available
			mains.Present |= capabilities.Voltage
			mains.Voltage = voltage
		}

		if pcResp[power_configuration.MainsFrequency].Status == 0 {
			frequency := float64(pcResp[power_configuration.MainsFrequency].DataTypeValue.Value.(uint64)) / 2.0

			mains.Present |= capabilities.Available
			mains.Present |= capabilities.Frequency
			mains.Frequency = frequency
		}

		if pcResp[power_configuration.BatteryVoltage].Status == 0 {
			voltage := float64(pcResp[power_configuration.BatteryVoltage].DataTypeValue.Value.(uint64)) / 10.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.Voltage
			battery.Voltage = voltage
		}

		if pcResp[power_configuration.BatteryPercentageRemaining].Status == 0 {
			remaining := float64(pcResp[power_configuration.BatteryPercentageRemaining].DataTypeValue.Value.(uint64)) / 2.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.Remaining
			battery.Remaining = remaining
		}

		if pcResp[power_configuration.BatteryRatedVoltage].Status == 0 {
			voltage := float64(pcResp[power_configuration.BatteryRatedVoltage].DataTypeValue.Value.(uint64)) / 10.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.NominalVoltage
			battery.NominalVoltage = voltage
		}
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[d.Identifier]

	hasCapability := mains.Available || battery.Available

	if hasCapability {
		powerStatus := capabilities.PowerStatus{}

		if mains.Available {
			powerStatus.Mains = []capabilities.PowerMainsStatus{mains}
		}

		if battery.Available {
			powerStatus.Battery = []capabilities.PowerBatteryStatus{battery}
		}

		data.PowerStatus = powerStatus

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.PowerSupplyFlag)
	} else {
		data.PowerStatus = capabilities.PowerStatus{}

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.PowerSupplyFlag)
	}

	i.data[d.Identifier] = data

	return nil
}
