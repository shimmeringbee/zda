package power_supply

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability/proprietary/xiaomi"
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

	i.attMonMainsVoltage.Detach(ctx, d)
	i.attMonMainsFrequency.Detach(ctx, d)
	i.attMonBatteryVoltage.Detach(ctx, d)
	i.attMonBatteryPercentageRemaining.Detach(ctx, d)
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

	mains := capabilities.PowerMainsStatus{}
	battery := capabilities.PowerBatteryStatus{}

	basicEndpoints := zda.FindEndpointsWithClusterID(d, zcl.BasicId)
	if cfg.Bool("HasBasicPowerSource", len(basicEndpoints) > 0) {
		basicEndpoint := zigbee.Endpoint(cfg.Int("BasicEndpoint", int(selectEndpoint(basicEndpoints, d.Endpoints))))

		i.supervisor.Logger().LogInfo(ctx, "Power Supply capability has Basic support.", logwrap.Datum("Endpoint", basicEndpoint))

		i.supervisor.Logger().LogDebug(ctx, "Reading power source information from Basic cluster.")
		basicResp, err := i.supervisor.ZCL().ReadAttributes(ctx, d, basicEndpoint, zcl.BasicId, []zcl.AttributeID{basic.PowerSource})
		if err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to read power source information from basic cluster.", logwrap.Err(err))
			return err
		}

		if basicResp[basic.PowerSource].Status == 0 {
			var value uint64
			switch basicResp[basic.PowerSource].DataTypeValue.DataType {
			case zcl.TypeEnum8:
				value = uint64(basicResp[basic.PowerSource].DataTypeValue.Value.(uint8))
			case zcl.TypeUnsignedInt8:
				value = basicResp[basic.PowerSource].DataTypeValue.Value.(uint64)
			}

			if (value&0x80) == 0x80 || (value&0x7f) == 0x03 {
				battery.Available = true
				battery.Present |= capabilities.Available
			}

			if (value & 0x7f) != 0x03 {
				mains.Available = true
				mains.Present |= capabilities.Available
			}

			i.supervisor.Logger().LogInfo(ctx, "Basic cluster returned status.", logwrap.Datum("PowerSource", basicResp[basic.PowerSource].Status))
		} else {
			i.supervisor.Logger().LogWarn(ctx, "Basic cluster errored.", logwrap.Datum("Status", basicResp[basic.PowerSource].Status))
		}
	}

	needsPolling := false

	pcEndpoints := zda.FindEndpointsWithClusterID(d, zcl.PowerConfigurationId)
	pcEndpoint := zigbee.Endpoint(cfg.Int("PowerConfigurationEndpoint", int(selectEndpoint(pcEndpoints, d.Endpoints))))

	hasPowerConfiguration := false

	if cfg.Bool("HasPowerConfiguration", len(pcEndpoints) > 0) {
		hasPowerConfiguration = true

		i.supervisor.Logger().LogInfo(ctx, "Power Supply capability has PowerConfiguration support.", logwrap.Datum("Endpoint", pcEndpoint))

		i.supervisor.Logger().LogDebug(ctx, "Reading mains and battery attributes from PowerConfiguration.")
		pcResp, err := i.supervisor.ZCL().ReadAttributes(ctx, d, pcEndpoint, zcl.PowerConfigurationId, []zcl.AttributeID{power_configuration.MainsVoltage, power_configuration.MainsFrequency, power_configuration.BatteryVoltage, power_configuration.BatteryPercentageRemaining, power_configuration.BatteryRatedVoltage})
		if err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to read PowerConfiguration attributes.", logwrap.Err(err))
			return err
		}

		if pcResp[power_configuration.MainsVoltage].Status == 0 {
			i.supervisor.Logger().LogInfo(ctx, "PowerConfiguration supports Mains Voltage.")
			voltage := float64(pcResp[power_configuration.MainsVoltage].DataTypeValue.Value.(uint64)) / 10.0

			mains.Present |= capabilities.Available
			mains.Present |= capabilities.Voltage
			mains.Voltage = voltage

			reportableChange := uint(cfg.Int("MainVoltageReportableChange", 1))
			if polling, err := i.attMonMainsVoltage.Attach(ctx, d, pcEndpoint, reportableChange); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Mains Voltage attribute monitor to device.", logwrap.Err(err))
				return err
			} else if polling {
				needsPolling = polling
			}
		}

		if pcResp[power_configuration.MainsFrequency].Status == 0 {
			i.supervisor.Logger().LogInfo(ctx, "PowerConfiguration supports Mains Frequency.")
			frequency := float64(pcResp[power_configuration.MainsFrequency].DataTypeValue.Value.(uint64)) / 2.0

			mains.Present |= capabilities.Available
			mains.Present |= capabilities.Frequency
			mains.Frequency = frequency

			reportableChange := uint(cfg.Int("MainsFrequencyReportableChange", 1))
			if polling, err := i.attMonMainsFrequency.Attach(ctx, d, pcEndpoint, reportableChange); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Mains Frequency attribute monitor to device.", logwrap.Err(err))
				return err
			} else if polling {
				needsPolling = polling
			}
		}

		if pcResp[power_configuration.BatteryVoltage].Status == 0 {
			i.supervisor.Logger().LogInfo(ctx, "PowerConfiguration supports Battery Voltage.")
			voltage := float64(pcResp[power_configuration.BatteryVoltage].DataTypeValue.Value.(uint64)) / 10.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.Voltage
			battery.Voltage = voltage

			reportableChange := uint(cfg.Int("BatteryVoltageReportableChange", 1))
			if polling, err := i.attMonBatteryVoltage.Attach(ctx, d, pcEndpoint, reportableChange); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Battery Voltage attribute monitor to device.", logwrap.Err(err))
				return err
			} else if polling {
				needsPolling = polling
			}
		}

		if pcResp[power_configuration.BatteryPercentageRemaining].Status == 0 {
			i.supervisor.Logger().LogInfo(ctx, "PowerConfiguration supports Battery Percentage Remaining.")
			remaining := float64(pcResp[power_configuration.BatteryPercentageRemaining].DataTypeValue.Value.(uint64)) / 200.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.Remaining
			battery.Remaining = remaining

			reportableChange := uint(cfg.Int("BatteryPercentageRemainingReportableChange", 1))
			if polling, err := i.attMonBatteryPercentageRemaining.Attach(ctx, d, pcEndpoint, reportableChange); err != nil {
				i.supervisor.Logger().LogError(ctx, "Failed to attach Battery Percentage Remaining attribute monitor to device.", logwrap.Err(err))
				return err
			} else if polling {
				needsPolling = polling
			}
		}

		if pcResp[power_configuration.BatteryRatedVoltage].Status == 0 {
			i.supervisor.Logger().LogInfo(ctx, "PowerConfiguration supports Battery Rated Voltage.")
			voltage := float64(pcResp[power_configuration.BatteryRatedVoltage].DataTypeValue.Value.(uint64)) / 10.0

			battery.Present |= capabilities.Available
			battery.Present |= capabilities.MaximumVoltage
			battery.MaximumVoltage = voltage
		}

		if needsPolling {
			i.supervisor.Logger().LogDebug(ctx, "PowerConfiguration needs polling support.")
		}
	}

	hasVendorXiaomiApproachOne := false

	if cfg.Bool("HasVendorXiaomiApproachOne", false) {
		hasVendorXiaomiApproachOne = true

		i.supervisor.Logger().LogInfo(ctx, "Power Supply capability has Vendor Xiaomi Approach One support.", logwrap.Datum("Endpoint", pcEndpoint))

		battery.Available = true
		battery.Present |= capabilities.Available
		battery.Present |= capabilities.Voltage

		if polling, err := i.attMonVendorXiaomiApproachOne.Attach(ctx, d, pcEndpoint, nil); err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to attach Vendor Xiaomi Approach One Battery Voltage attribute monitor to device.", logwrap.Err(err))
			return err
		} else if polling {
			needsPolling = polling
		}
	}

	if battery.Available {
		battery.MinimumVoltage = cfg.Float("MinimumVoltage", battery.MinimumVoltage)
		if battery.MinimumVoltage > 0 {
			battery.Present |= capabilities.MinimumVoltage
		}

		battery.MaximumVoltage = cfg.Float("MaximumVoltage", battery.MaximumVoltage)
		if battery.MaximumVoltage > 0 {
			battery.Present |= capabilities.MaximumVoltage
		}
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[d.Identifier]

	hasCapability := mains.Available || battery.Available

	if hasCapability {
		if mains.Available {
			data.Mains = []*capabilities.PowerMainsStatus{&mains}
		}

		if battery.Available {
			data.Battery = []*capabilities.PowerBatteryStatus{&battery}
		}

		data.RequiresPolling = needsPolling
		data.Endpoint = pcEndpoint
		data.PowerConfiguration = hasPowerConfiguration
		data.VendorXiaomiApproachOne = hasVendorXiaomiApproachOne

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.PowerSupplyFlag)
	} else {
		data.Mains = nil
		data.Battery = nil

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.PowerSupplyFlag)

		i.attMonMainsVoltage.Detach(ctx, d)
		i.attMonMainsFrequency.Detach(ctx, d)
		i.attMonBatteryVoltage.Detach(ctx, d)
		i.attMonBatteryPercentageRemaining.Detach(ctx, d)
		i.attMonVendorXiaomiApproachOne.Detach(ctx, d)
	}

	i.data[d.Identifier] = data

	return nil
}

func (i *Implementation) attributeUpdateMainsVoltage(device zda.Device, id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[device.Identifier]
	if len(data.Mains) > 0 && (data.Mains[0].Present&capabilities.Voltage) == capabilities.Voltage {
		newVoltage := float64(value.Value.(uint64)) / 10.0
		data.LastUpdateTime = time.Now()

		if newVoltage != data.Mains[0].Voltage {
			data.Mains[0].Voltage = newVoltage
			data.LastChangeTime = data.LastUpdateTime
		}

		i.data[device.Identifier] = data

		i.supervisor.Logger().LogDebug(context.Background(), "Mains voltage update received.", logwrap.Datum("MainsVoltage", data.Mains[0].Voltage), logwrap.Datum("Identifier", device.Identifier.String()))
	}
}

func (i *Implementation) attributeUpdateMainsFrequency(device zda.Device, id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[device.Identifier]
	if len(data.Mains) > 0 && (data.Mains[0].Present&capabilities.Frequency) == capabilities.Frequency {
		newFrequency := float64(value.Value.(uint64)) / 2.0
		data.LastUpdateTime = time.Now()

		if newFrequency != data.Mains[0].Frequency {
			data.Mains[0].Frequency = newFrequency
			data.LastChangeTime = data.LastUpdateTime
		}

		i.data[device.Identifier] = data

		i.supervisor.Logger().LogDebug(context.Background(), "Mains frequency update received.", logwrap.Datum("MainsFrequency", data.Mains[0].Frequency), logwrap.Datum("Identifier", device.Identifier.String()))
	}
}

func (i *Implementation) attributeUpdateBatteryVoltage(device zda.Device, id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[device.Identifier]
	if len(data.Battery) > 0 && (data.Battery[0].Present&capabilities.Voltage) == capabilities.Voltage {
		newVoltage := float64(value.Value.(uint64)) / 10.0
		data.LastUpdateTime = time.Now()

		if newVoltage != data.Battery[0].Voltage {
			data.Battery[0].Voltage = newVoltage
			data.LastChangeTime = data.LastUpdateTime
		}

		i.data[device.Identifier] = data

		i.supervisor.Logger().LogDebug(context.Background(), "Battery voltage update received.", logwrap.Datum("BatteryVoltage", data.Battery[0].Voltage), logwrap.Datum("Identifier", device.Identifier.String()))
	}
}

func (i *Implementation) attributeUpdateBatterPercentageRemaining(device zda.Device, id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[device.Identifier]
	if len(data.Battery) > 0 && (data.Battery[0].Present&capabilities.Remaining) == capabilities.Remaining {
		newRemaining := float64(value.Value.(uint64)) / 200.0
		data.LastUpdateTime = time.Now()

		if newRemaining != data.Battery[0].Remaining {
			data.Battery[0].Remaining = newRemaining
			data.LastChangeTime = data.LastUpdateTime
		}

		i.data[device.Identifier] = data

		i.supervisor.Logger().LogDebug(context.Background(), "Battery percentage remaining update received.", logwrap.Datum("BatteryPercentageRemaining", data.Battery[0].Remaining), logwrap.Datum("Identifier", device.Identifier.String()))
	}
}

func (i *Implementation) attributeUpdateVendorXiaomiApproachOne(device zda.Device, id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	data := i.data[device.Identifier]
	if len(data.Battery) > 0 && (data.Battery[0].Present&capabilities.Voltage) == capabilities.Voltage {
		xal, err := xiaomi.ParseAttributeList([]byte(value.Value.(string)))
		if err != nil {
			i.supervisor.Logger().LogError(context.Background(), "Failed to parse Xiaomi attribute list.", logwrap.Datum("Identifier", device.Identifier.String()), logwrap.Err(err))
			return
		}

		att, found := xal[1]

		if found {
			newVoltage := float64(att.Attribute.Value.(uint64)) / 1000.0
			data.LastUpdateTime = time.Now()

			if newVoltage != data.Battery[0].Voltage {
				data.Battery[0].Voltage = newVoltage
				data.LastChangeTime = data.LastUpdateTime
			}

			i.data[device.Identifier] = data

			i.supervisor.Logger().LogDebug(context.Background(), "Battery voltage update received, Xiaomi approach one.", logwrap.Datum("BatteryVoltage", data.Battery[0].Voltage), logwrap.Datum("Identifier", device.Identifier.String()))
		}
	}
}
