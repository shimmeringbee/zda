package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
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

		i.supervisor.Logger().LogInfo(ctx, "Have Product Information capability.", logwrap.Datum("Endpoint", endpoint))

		var productData ProductData

		i.supervisor.Logger().LogDebug(ctx, "Querying Basic cluster for name and model.")
		records, err := i.supervisor.ZCL().ReadAttributes(ctx, d, endpoint, zcl.BasicId, []zcl.AttributeID{basic.ManufacturerName, basic.ModelIdentifier})
		if err != nil {
			i.supervisor.Logger().LogError(ctx, "Failed to query Basic cluster for details", logwrap.Err(err))
			return err
		}

		if records[basic.ManufacturerName].Status == 0 {
			manufacturerString := records[basic.ManufacturerName].DataTypeValue.Value.(string)
			productData.Manufacturer = &manufacturerString
			i.supervisor.Logger().LogInfo(ctx, "Manufacturer name retrieved.", logwrap.Datum("Name", manufacturerString))
		}

		if records[basic.ModelIdentifier].Status == 0 {
			productString := records[basic.ModelIdentifier].DataTypeValue.Value.(string)
			productData.Product = &productString
			i.supervisor.Logger().LogInfo(ctx, "Product name retrieved.", logwrap.Datum("Name", productString))
		}

		i.datalock.Lock()
		i.data[d.Identifier] = productData
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.HasProductInformationFlag)
	}

	return nil
}
