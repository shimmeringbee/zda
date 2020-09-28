package has_product_information

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/capability"
)

type addedDeviceReq struct {
	device capability.Device
	ch     chan error
}

type removedDeviceReq struct {
	device capability.Device
	ch     chan error
}

type enumerateDeviceReq struct {
	device capability.Device
	ch     chan error
}

type enumerateDeviceComplete struct {
	device      capability.Device
	productData ProductData
	ch          chan error
}

type productInformationReq struct {
	device capability.Device
	ch     chan productInformationResp
}

type productInformationResp struct {
	ProductInformation capabilities.ProductInformation
	error              error
}
