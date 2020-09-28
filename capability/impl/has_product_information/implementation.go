package has_product_information

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zigbee"
	"log"
)

type ZigbeeHasProductInformation struct {
	gateway               da.Gateway
	nodeTable             nodeTable
	internalCallbacks     callbacks.Adder
	zclGlobalCommunicator zclGlobalCommunicator
	capabilityManager     CapabilityManager
}

func (z *ZigbeeHasProductInformation) Capability() da.Capability {
	return capabilities.HasProductInformationFlag
}

func (z *ZigbeeHasProductInformation) Init() {
	z.internalCallbacks.Add(z.NodeEnumerationCallback)
}

func (z *ZigbeeHasProductInformation) NodeEnumerationCallback(ctx context.Context, ide internalDeviceEnumeration) error {
	iDev := ide.device
	iNode := iDev.node

	iDev.mutex.Lock()

	found := false
	var foundEndpoint zigbee.Endpoint

	for _, endpoint := range iDev.endpoints {
		if isClusterIdInSlice(iNode.endpointDescriptions[endpoint].InClusterList, zcl.BasicId) {
			found = true
			foundEndpoint = endpoint
			break
		}
	}

	if found {
		if err := retry.Retry(ctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
			readRecords, err := z.zclGlobalCommunicator.ReadAttributes(ctx, iNode.ieeeAddress, iNode.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, foundEndpoint, iNode.nextTransactionSequence(), []zcl.AttributeID{0x0004, 0x0005})

			if err == nil {
				for _, record := range readRecords {
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

			return err
		}); err != nil {
			log.Printf("failed to read product information: %s", err)
		}

		iDev.mutex.Unlock()
		z.capabilityManager.AddCapabilityToDevice(iDev.generateIdentifier(), capabilities.HasProductInformationFlag)
	} else {
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

	iDev := z.nodeTable.getDevice(device.Identifier().(IEEEAddressWithSubIdentifier))
	if iDev != nil {
		iDev.mutex.RLock()
		defer iDev.mutex.RUnlock()

		return iDev.productInformation, nil
	} else {
		return capabilities.ProductInformation{}, fmt.Errorf("internal node table error, missing device")
	}
}
