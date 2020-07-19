package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"log"
)

type ZigbeeOnOffState struct {
	State bool
}

type ZigbeeOnOff struct {
	Gateway da.Gateway

	addInternalCallback addInternalCallback
	deviceStore         deviceStore
	nodeStore           nodeStore

	zclCommunicatorCallbacks zclCommunicatorCallbacks
	zclCommunicatorRequests  zclCommunicatorRequests
	zclGlobalCommunicator    zclGlobalCommunicator

	nodeBinder zigbee.NodeBinder
}

func (z *ZigbeeOnOff) Init() {
	z.addInternalCallback(z.NodeEnumerationCallback)

	z.zclCommunicatorCallbacks.AddCallback(z.zclCommunicatorCallbacks.NewMatch(func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
		_, canCast := zclMessage.Command.(*global.ReportAttributes)
		return zclMessage.ClusterID == zcl.OnOffId && canCast
	}, z.incomingReportAttributes))
}

func (z *ZigbeeOnOff) NodeEnumerationCallback(ctx context.Context, ine internalNodeEnumeration) error {
	node := ine.node

	node.mutex.Lock()
	defer node.mutex.Unlock()

	for _, dev := range node.devices {
		dev.mutex.Lock()

		if endpoint, found := findEndpointWithClusterId(node, dev, zcl.OnOffId); found {
			addCapability(&dev.device, capabilities.OnOffFlag)

			if err := retry.Retry(ctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
				return z.nodeBinder.BindNodeToController(ctx, node.ieeeAddress, endpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId)
			}); err != nil {
				log.Printf("failed to bind to zda: %s", err)
			}

			if err := retry.Retry(ctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
				return z.zclGlobalCommunicator.ConfigureReporting(ctx, node.ieeeAddress, node.supportsAPSAck, zcl.OnOffId, zigbee.NoManufacturer, endpoint, DefaultGatewayHomeAutomationEndpoint, node.nextTransactionSequence(), onoff.OnOff, zcl.TypeBoolean, 0, 60, nil)
			}); err != nil {
				log.Printf("failed to configure reporting to zda: %s", err)
			}
		} else {
			removeCapability(&dev.device, capabilities.OnOffFlag)
		}

		dev.mutex.Unlock()
	}

	return nil
}

func (z *ZigbeeOnOff) On(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.Gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.OnOffFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDevice, found := z.deviceStore.getDevice(device.Identifier)

	if !found {
		return fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}

	iNode := iDevice.node

	iNode.mutex.RLock()
	defer iNode.mutex.RUnlock()

	iDevice.mutex.RLock()
	defer iDevice.mutex.RUnlock()

	endpoint, found := findEndpointWithClusterId(iNode, iDevice, zcl.OnOffId)

	if !found {
		return fmt.Errorf("unable to find on off cluster on zigbee device in zda")
	}

	zclMsg := zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: iNode.nextTransactionSequence(),
		Manufacturer:        0,
		ClusterID:           zcl.OnOffId,
		SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
		DestinationEndpoint: endpoint,
		Command:             &onoff.On{},
	}

	return z.zclCommunicatorRequests.Request(ctx, iNode.ieeeAddress, iNode.supportsAPSAck, zclMsg)
}

func (z *ZigbeeOnOff) Off(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.Gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.OnOffFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDevice, found := z.deviceStore.getDevice(device.Identifier)

	if !found {
		return fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}

	iNode := iDevice.node

	iNode.mutex.RLock()
	defer iNode.mutex.RUnlock()

	iDevice.mutex.RLock()
	defer iDevice.mutex.RUnlock()

	endpoint, found := findEndpointWithClusterId(iNode, iDevice, zcl.OnOffId)

	if !found {
		return fmt.Errorf("unable to find on off cluster on zigbee device in zda")
	}

	zclMsg := zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: iNode.nextTransactionSequence(),
		Manufacturer:        0,
		ClusterID:           zcl.OnOffId,
		SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
		DestinationEndpoint: endpoint,
		Command:             &onoff.Off{},
	}

	return z.zclCommunicatorRequests.Request(ctx, iNode.ieeeAddress, iNode.supportsAPSAck, zclMsg)
}

func (z *ZigbeeOnOff) State(ctx context.Context, device da.Device) (bool, error) {
	if da.DeviceDoesNotBelongToGateway(z.Gateway, device) {
		return false, da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.OnOffFlag) {
		return false, da.DeviceDoesNotHaveCapability
	}

	iDevice, found := z.deviceStore.getDevice(device.Identifier)

	if !found {
		return false, fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}

	iDevice.mutex.RLock()
	defer iDevice.mutex.RUnlock()

	return iDevice.onOffState.State, nil
}

func (z *ZigbeeOnOff) incomingReportAttributes(source communicator.MessageWithSource) {
	node, found := z.nodeStore.getNode(source.SourceAddress)

	if !found {
		return
	}

	report := source.Message.Command.(*global.ReportAttributes)

	node.mutex.RLock()
	defer node.mutex.RUnlock()

	for _, device := range node.devices {
		device.mutex.Lock()

		if isEndpointInSlice(device.endpoints, source.Message.SourceEndpoint) {
			if device.device.HasCapability(capabilities.OnOffFlag) {
				for _, attributeReport := range report.Records {
					switch attributeReport.Identifier {
					case onoff.OnOff:
						state, ok := attributeReport.DataTypeValue.Value.(bool)

						if ok {
							device.onOffState.State = state
						}
					}
				}
			}
		}

		device.mutex.Unlock()
	}
}
