package power_supply

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

func (i *Implementation) Status(ctx context.Context, dad da.Device) (capabilities.PowerStatus, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return capabilities.PowerStatus{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.PowerSupplyFlag) {
		return capabilities.PowerStatus{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	var resMains []capabilities.PowerMainsStatus

	for _, mains := range i.data[d.Identifier].Mains {
		resMains = append(resMains, *mains)
	}

	var resBattery []capabilities.PowerBatteryStatus

	for _, battery := range i.data[d.Identifier].Battery {
		copiedBattery := *battery

		if (copiedBattery.Present & capabilities.Remaining) != capabilities.Remaining {
			required := capabilities.Voltage | capabilities.MinimumVoltage | capabilities.MaximumVoltage

			if (copiedBattery.Present & required) == required {
				copiedBattery.Present |= capabilities.Remaining

				currentValue := copiedBattery.Voltage
				if currentValue > copiedBattery.MaximumVoltage {
					currentValue = copiedBattery.MaximumVoltage
				}

				if currentValue < copiedBattery.MinimumVoltage {
					currentValue = copiedBattery.MinimumVoltage
				}

				copiedBattery.Remaining = (currentValue - copiedBattery.MinimumVoltage) / (copiedBattery.MaximumVoltage - copiedBattery.MinimumVoltage)
			}
		}

		resBattery = append(resBattery, copiedBattery)
	}

	return capabilities.PowerStatus{
		Mains:   resMains,
		Battery: resBattery,
	}, nil
}
