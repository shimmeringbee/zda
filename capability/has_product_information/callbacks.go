package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
)

func (i *Implementation) addedDeviceCallback(ctx context.Context, e zda.AddedDeviceEvent) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[e.Device.Identifier]; !found {
		i.data[e.Device.Identifier] = ProductData{}
	}

	return nil
}

func (i *Implementation) removedDeviceCallback(ctx context.Context, e zda.RemovedDeviceEvent) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	delete(i.data, e.Device.Identifier)

	return nil
}

func (i *Implementation) enumerateDeviceCallback(ctx context.Context, e zda.EnumerateDeviceEvent) error {
	endpoints := zda.FindEndpointsWithClusterID(e.Device, zcl.BasicId)

	if len(endpoints) == 0 {
		i.datalock.Lock()
		i.data[e.Device.Identifier] = ProductData{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(e.Device, capabilities.HasProductInformationFlag)
	} else {
		endpoint := endpoints[0]

		var productData ProductData

		records, err := i.supervisor.ZCL().ReadAttributes(ctx, e.Device, endpoint, zcl.BasicId, []zcl.AttributeID{0x0004, 0x0005})
		if err != nil {
			return err
		}

		if records[0x0004].Status == 0 {
			productData.Manufacturer = records[0x0004].DataTypeValue.Value.(*string)
		}

		if records[0x0005].Status == 0 {
			productData.Product = records[0x0005].DataTypeValue.Value.(*string)
		}

		i.datalock.Lock()
		i.data[e.Device.Identifier] = productData
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(e.Device, capabilities.HasProductInformationFlag)
	}

	return nil
}
