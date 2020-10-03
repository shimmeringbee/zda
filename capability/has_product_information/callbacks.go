package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
)

func (i *Implementation) AddedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[d.Identifier]; !found {
		i.data[d.Identifier] = ProductData{}
	}

	return nil
}

func (i *Implementation) RemovedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	delete(i.data, d.Identifier)

	return nil
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	endpoints := zda.FindEndpointsWithClusterID(d, zcl.BasicId)

	if len(endpoints) == 0 {
		i.datalock.Lock()
		i.data[d.Identifier] = ProductData{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.HasProductInformationFlag)
	} else {
		endpoint := endpoints[0]

		var productData ProductData

		records, err := i.supervisor.ZCL().ReadAttributes(ctx, d, endpoint, zcl.BasicId, []zcl.AttributeID{0x0004, 0x0005})
		if err != nil {
			return err
		}

		if records[0x0004].Status == 0 {
			manufacturerString := records[0x0004].DataTypeValue.Value.(string)
			productData.Manufacturer = &manufacturerString
		}

		if records[0x0005].Status == 0 {
			productString := records[0x0005].DataTypeValue.Value.(string)
			productData.Product = &productString
		}

		i.datalock.Lock()
		i.data[d.Identifier] = productData
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.HasProductInformationFlag)
	}

	return nil
}
