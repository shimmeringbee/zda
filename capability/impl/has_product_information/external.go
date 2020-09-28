package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
)

func (i *Implementation) ProductInformation(ctx context.Context, device da.Device) (capabilities.ProductInformation, error)  {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return capabilities.ProductInformation{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.HasProductInformationFlag) {
		return capabilities.ProductInformation{}, da.DeviceDoesNotHaveCapability
	}

	ch := make(chan productInformationResp, 1)
	i.msgCh <- productInformationReq{device: d, ch: ch}

	select {
	case resp := <-ch:
		return resp.ProductInformation, resp.error
	case <-ctx.Done():
		return capabilities.ProductInformation{}, zigbee.ContextExpired
	}
}
