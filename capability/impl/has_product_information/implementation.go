package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
)

type ProductData struct {
	Manufacturer *string
	Product      *string
}

type Implementation struct {
	stopCh chan struct{}
	msgCh  chan interface{}
	data   map[zda.IEEEAddressWithSubIdentifier]ProductData

	supervisor capability.Supervisor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.HasProductInformationFlag
}

func (i *Implementation) Init(supervisor capability.Supervisor) {
	i.stopCh = make(chan struct{})
	i.msgCh = make(chan interface{})
	i.data = make(map[zda.IEEEAddressWithSubIdentifier]ProductData)
	i.supervisor = supervisor

	supervisor.EventSubscription().AddedDevice(i.addedDeviceCallback)
	supervisor.EventSubscription().RemovedDevice(i.removedDeviceCallback)
	supervisor.EventSubscription().EnumerateDevice(i.enumerateDeviceCallback)
}

func (i *Implementation) Start() {
	go i.loop()
}

func (i *Implementation) Stop() {
	i.stopCh <- struct{}{}
}

func (i *Implementation) loop() {
	for {
		select {
		case <-i.stopCh:
			return
		case m := <-i.msgCh:
			i.handle(m)
		}
	}
}

func (i *Implementation) handle(m interface{}) {
	switch e := m.(type) {
	default:
	case addedDeviceReq:
		i.handleAddedDeviceRequest(e)
	case removedDeviceReq:
		i.handleRemovedDeviceRequest(e)
	case productInformationReq:
		i.handleProductInformationReq(e)
	case enumerateDeviceReq:
		i.handleEnumerateDeviceReq(e)
	case enumerateDeviceComplete:
		i.handleEnumerateDeviceComplete(e)
	}
}

func (i *Implementation) handleAddedDeviceRequest(e addedDeviceReq) {
	if _, found := i.data[e.device.Identifier]; !found {
		i.data[e.device.Identifier] = ProductData{}
	}

	e.ch <- nil
}

func (i *Implementation) handleRemovedDeviceRequest(e removedDeviceReq) {
	delete(i.data, e.device.Identifier)

	e.ch <- nil
}

func (i *Implementation) handleProductInformationReq(e productInformationReq) {
	var resp productInformationResp

	if data, found := i.data[e.device.Identifier]; found {
		if data.Manufacturer != nil {
			resp.ProductInformation.Manufacturer = *data.Manufacturer
			resp.ProductInformation.Present |= capabilities.Manufacturer
		}

		if data.Product != nil {
			resp.ProductInformation.Name = *data.Product
			resp.ProductInformation.Present |= capabilities.Name
		}

	} else {
		resp.error = da.DeviceDoesNotBelongToGatewayError
	}

	e.ch <- resp
}

func (i *Implementation) handleEnumerateDeviceReq(e enumerateDeviceReq) {
	endpoints := capability.FindEndpointsWithClusterID(e.device, zcl.BasicId)

	if len(endpoints) == 0 {
		i.data[e.device.Identifier] = ProductData{}
		i.supervisor.ManageDeviceCapabilities().Remove(e.device, capabilities.HasProductInformationFlag)
		e.ch <- nil
	} else {
		endpoint := endpoints[0]

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), zda.DefaultNetworkTimeout*zda.DefaultNetworkRetries)
			defer cancel()

			var productData ProductData

			records, err := i.supervisor.ZCL().ReadAttributes(ctx, e.device, endpoint, zcl.BasicId, []zcl.AttributeID{0x0004, 0x0005})
			if err != nil {
				e.ch <- err
				return
			}

			if records[0x0004].Status == 0 {
				productData.Manufacturer = records[0x0004].DataTypeValue.Value.(*string)
			}

			if records[0x0005].Status == 0 {
				productData.Product = records[0x0005].DataTypeValue.Value.(*string)
			}

			i.msgCh <- enumerateDeviceComplete{
				device:      e.device,
				productData: productData,
				ch:          e.ch,
			}
		}()
	}
}

func (i *Implementation) handleEnumerateDeviceComplete(e enumerateDeviceComplete) {
	i.supervisor.ManageDeviceCapabilities().Add(e.device, capabilities.HasProductInformationFlag)
	i.data[e.device.Identifier] = e.productData
	e.ch <- nil
}
