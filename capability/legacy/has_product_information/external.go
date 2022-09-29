package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

var _ capabilities.HasProductInformation = (*Implementation)(nil)

func (i *Implementation) ProductInformation(ctx context.Context, device da.Device) (capabilities.ProductInformation, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return capabilities.ProductInformation{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.HasProductInformationFlag) {
		return capabilities.ProductInformation{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	data, found := i.data[d.Identifier]
	i.datalock.RUnlock()

	var ret capabilities.ProductInformation
	var err error

	if found {
		if data.Manufacturer != nil {
			ret.Manufacturer = *data.Manufacturer
			ret.Present |= capabilities.Manufacturer
		}

		if data.Product != nil {
			ret.Name = *data.Product
			ret.Present |= capabilities.Name
		}
	} else {
		err = da.DeviceDoesNotBelongToGatewayError
	}

	return ret, err
}
