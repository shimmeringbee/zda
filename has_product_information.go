package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
)

type ZigbeeHasProductInformation struct {
	gateway *ZigbeeGateway
}

func (z *ZigbeeHasProductInformation) Init() {
	z.gateway.callbacks.Add(z.NodeEnumerationCallback)
}

func (z *ZigbeeHasProductInformation) NodeEnumerationCallback(ctx context.Context, ine internalNodeEnumeration) error {
	ine.node.mutex.RLock()
	defer ine.node.mutex.RUnlock()

	for _, iDev := range ine.node.devices {
		iDev.mutex.Lock()

		found := false
		foundEndpoint := zigbee.Endpoint(0x0000)

		for _, endpoint := range iDev.endpoints {
			if isClusterIdInSlice(ine.node.endpointDescriptions[endpoint].InClusterList, 0x0000) {
				found = true
				foundEndpoint = endpoint
				break
			}
		}

		if found {
			request := zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ClientToServer,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           0x0000,
				SourceEndpoint:      1,
				DestinationEndpoint: foundEndpoint,
				Command: &global.ReadAttributes{
					Identifier: []zcl.AttributeID{0x0004, 0x0005},
				},
			}

			response, err := z.gateway.communicator.RequestResponse(ctx, ine.node.ieeeAddress, request)

			if err != nil {
				iDev.mutex.Unlock()
				return err
			}

			readResponse, is := response.Command.(*global.ReadAttributesResponse)

			if is {
				for _, record := range readResponse.Records {
					switch record.Identifier {
					case 0x0004:
						if record.Status == 0 {
							iDev.productInformation.Manufacturer = record.DataTypeValue.Value.(string)
							iDev.productInformation.Present |= capabilities.Manufacturer
						} else {
							iDev.productInformation.Manufacturer = ""
							iDev.productInformation.Present &= ^capabilities.Manufacturer
						}

					case 0x0005:
						if record.Status == 0 {
							iDev.productInformation.Name = record.DataTypeValue.Value.(string)
							iDev.productInformation.Present |= capabilities.Name
						} else {
							iDev.productInformation.Name = ""
							iDev.productInformation.Present &= ^capabilities.Name
						}
					}
				}
			}

			if !isCapabilityInSlice(iDev.device.Capabilities, capabilities.HasProductInformationFlag) {
				iDev.device.Capabilities = append(iDev.device.Capabilities, capabilities.HasProductInformationFlag)
			}
		}

		iDev.mutex.Unlock()
	}

	return nil
}

func (z *ZigbeeHasProductInformation) ProductInformation(ctx context.Context, device da.Device) (capabilities.ProductInformation, error) {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return capabilities.ProductInformation{}, da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.HasProductInformationFlag) {
		return capabilities.ProductInformation{}, da.DeviceDoesNotHaveCapability
	}

	iDev, _ := z.gateway.getDevice(device.Identifier)

	iDev.mutex.RLock()
	defer iDev.mutex.RUnlock()

	return iDev.productInformation, nil
}
