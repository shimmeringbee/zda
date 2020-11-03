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

	return i.data[d.Identifier].PowerStatus, nil
}
