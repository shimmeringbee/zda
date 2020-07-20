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
	"time"
)

type ZigbeeOnOffState struct {
	State           bool
	requiresPolling bool
}

type ZigbeeOnOff struct {
	Gateway da.Gateway

	addInternalCallback addInternalCallback
	deviceStore         deviceStore
	nodeStore           nodeStore

	zclCommunicatorCallbacks zclCommunicatorCallbacks
	zclCommunicatorRequests  zclCommunicatorRequests
	zclGlobalCommunicator    zclGlobalCommunicator

	nodeBinder  zigbee.NodeBinder
	poller      poller
	eventSender eventSender
}

const pollInterval = 5 * time.Second
const delayAfterSetForPolling = 500 * time.Millisecond

func (z *ZigbeeOnOff) Init() {
	z.addInternalCallback(z.NodeEnumerationCallback)
	z.addInternalCallback(z.NodeJoinCallback)

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

		dev.onOffState.requiresPolling = false

		if endpoint, found := findEndpointWithClusterId(node, dev, zcl.OnOffId); found {
			addCapability(&dev.device, capabilities.OnOffFlag)

			if err := retry.Retry(ctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
				return z.nodeBinder.BindNodeToController(ctx, node.ieeeAddress, endpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId)
			}); err != nil {
				log.Printf("failed to bind to zda: %s", err)
				dev.onOffState.requiresPolling = true
			}

			if err := retry.Retry(ctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
				return z.zclGlobalCommunicator.ConfigureReporting(ctx, node.ieeeAddress, node.supportsAPSAck, zcl.OnOffId, zigbee.NoManufacturer, endpoint, DefaultGatewayHomeAutomationEndpoint, node.nextTransactionSequence(), onoff.OnOff, zcl.TypeBoolean, 0, 60, nil)
			}); err != nil {
				log.Printf("failed to configure reporting to zda: %s", err)
				dev.onOffState.requiresPolling = true
			}
		} else {
			removeCapability(&dev.device, capabilities.OnOffFlag)
		}

		dev.mutex.Unlock()
	}

	return nil
}

func (z *ZigbeeOnOff) NodeJoinCallback(ctx context.Context, join internalNodeJoin) error {
	z.poller.AddNode(join.node, pollInterval, z.pollNode)
	return nil
}

func (z *ZigbeeOnOff) sendCommand(ctx context.Context, device da.Device, command interface{}) error {
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
		Command:             command,
	}

	err := z.zclCommunicatorRequests.Request(ctx, iNode.ieeeAddress, iNode.supportsAPSAck, zclMsg)

	if err == nil && iDevice.onOffState.requiresPolling {
		time.AfterFunc(delayAfterSetForPolling, func() {
			ctx, done := context.WithTimeout(context.Background(), DefaultNetworkTimeout)
			defer done()
			z.pollDevice(ctx, iNode, iDevice)
		})
	}

	return err
}

func (z *ZigbeeOnOff) On(ctx context.Context, device da.Device) error {
	return z.sendCommand(ctx, device, &onoff.On{})
}

func (z *ZigbeeOnOff) Off(ctx context.Context, device da.Device) error {
	return z.sendCommand(ctx, device, &onoff.Off{})
}

func (z *ZigbeeOnOff) setState(device *internalDevice, newState bool) {
	device.onOffState.State = newState
	z.eventSender.sendEvent(capabilities.OnOffState{Device: device.device, State: newState})
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
							z.setState(device, state)
						}
					}
				}
			}
		}

		device.mutex.Unlock()
	}
}

func (z *ZigbeeOnOff) pollNode(pctx context.Context, iNode *internalNode) {
	iNode.mutex.RLock()
	defer iNode.mutex.RUnlock()

	for _, iDevice := range iNode.devices {
		z.pollDevice(pctx, iNode, iDevice)
	}
}

func (z *ZigbeeOnOff) pollDevice(pctx context.Context, iNode *internalNode, iDevice *internalDevice) {
	iDevice.mutex.RLock()

	if iDevice.device.HasCapability(capabilities.OnOffFlag) && iDevice.onOffState.requiresPolling && iNode.nodeDesc.LogicalType == zigbee.Router {
		endpoint, found := findEndpointWithClusterId(iNode, iDevice, zcl.OnOffId)
		iDevice.mutex.RUnlock()

		if found {
			if err := retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
				response, err := z.zclGlobalCommunicator.ReadAttributes(ctx, iNode.ieeeAddress, iNode.supportsAPSAck, zcl.OnOffId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, endpoint, iNode.nextTransactionSequence(), []zcl.AttributeID{onoff.OnOff})

				if err == nil && len(response) == 1 {
					state, ok := response[0].DataTypeValue.Value.(bool)

					if ok {
						iDevice.mutex.Lock()
						z.setState(iDevice, state)
						iDevice.mutex.Unlock()
					}
				}

				return err
			}); err != nil {
				log.Printf("failed to query on off state in zda: %s", err)
			}
		}
	} else {
		iDevice.mutex.RUnlock()
	}
}
