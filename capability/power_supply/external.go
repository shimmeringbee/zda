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
		resBattery = append(resBattery, *battery)
	}

	return capabilities.PowerStatus{
		Mains:   resMains,
		Battery: resBattery,
	}, nil
}
