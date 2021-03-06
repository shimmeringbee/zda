package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
)

type zclShim struct {
	nodeTable                nodeTable
	commandRegistry          *zcl.CommandRegistry
	zclGlobalCommunicator    zclGlobalCommunicator
	zigbeeNodeBinder         zigbee.NodeBinder
	zclCommunicatorRequests  zclCommunicatorRequests
	zclCommunicatorCallbacks zclCommunicatorCallbacks
}

func (s *zclShim) RegisterCommandLibrary(z ZCLCommandLibrary) {
	z(s.commandRegistry)
}

func (s *zclShim) ReadAttributes(pctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error) {
	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return map[zcl.AttributeID]global.ReadAttributeResponseRecord{}, da.DeviceDoesNotBelongToGatewayError
	}

	iDev.node.mutex.RLock()
	supportsAPSAck := iDev.node.supportsAPSAck
	iDev.node.mutex.RUnlock()

	returnRecords := map[zcl.AttributeID]global.ReadAttributeResponseRecord{}

	err := retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		records, err := s.zclGlobalCommunicator.ReadAttributes(ctx, iDev.node.ieeeAddress, supportsAPSAck, c, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, e, iDev.node.nextTransactionSequence(), a)

		if err == nil {
			for _, readRec := range records {
				returnRecords[readRec.Identifier] = readRec
			}
		}

		return err
	})

	return returnRecords, err
}

func (s *zclShim) WriteAttributes(pctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a map[zcl.AttributeID]zcl.AttributeDataTypeValue) (map[zcl.AttributeID]global.WriteAttributesResponseRecord, error) {
	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return nil, da.DeviceDoesNotBelongToGatewayError
	}

	iDev.node.mutex.RLock()
	supportsAPSAck := iDev.node.supportsAPSAck
	iDev.node.mutex.RUnlock()

	returnRecords := map[zcl.AttributeID]global.WriteAttributesResponseRecord{}
	for id := range a {
		returnRecords[id] = global.WriteAttributesResponseRecord{
			Status:     0,
			Identifier: id,
		}
	}

	err := retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		records, err := s.zclGlobalCommunicator.WriteAttributes(ctx, iDev.node.ieeeAddress, supportsAPSAck, c, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, e, iDev.node.nextTransactionSequence(), a)

		if err == nil {
			for _, readRec := range records {
				returnRecords[readRec.Identifier] = readRec
			}
		}

		return err
	})

	return returnRecords, err
}

func (s *zclShim) Bind(pctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID) error {
	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return da.DeviceDoesNotBelongToGatewayError
	}

	return retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		return s.zigbeeNodeBinder.BindNodeToController(ctx, iDev.node.ieeeAddress, DefaultGatewayHomeAutomationEndpoint, e, c)
	})
}

func (s *zclShim) ConfigureReporting(pctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, min uint16, max uint16, reportableChange interface{}) error {
	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return da.DeviceDoesNotBelongToGatewayError
	}

	iDev.node.mutex.RLock()
	supportsAPSAck := iDev.node.supportsAPSAck
	iDev.node.mutex.RUnlock()

	return retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		return s.zclGlobalCommunicator.ConfigureReporting(ctx, iDev.node.ieeeAddress, supportsAPSAck, c, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, e, iDev.node.nextTransactionSequence(), a, dt, min, max, reportableChange)
	})
}

func (s *zclShim) SendCommand(pctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, cmd interface{}) error {
	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return da.DeviceDoesNotBelongToGatewayError
	}

	iDev.node.mutex.RLock()
	supportsAPSAck := iDev.node.supportsAPSAck
	iDev.node.mutex.RUnlock()

	return retry.Retry(pctx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		return s.zclCommunicatorRequests.Request(ctx, iDev.node.ieeeAddress, supportsAPSAck, zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: iDev.node.nextTransactionSequence(),
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           c,
			SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
			DestinationEndpoint: e,
			Command:             cmd,
		})
	})
}

func (s *zclShim) Listen(f ZCLFilter, c ZCLCallback) {
	match := s.zclCommunicatorCallbacks.NewMatch(communicator.Matcher(f), func(source communicator.MessageWithSource) {
		iNode := s.nodeTable.getNode(source.SourceAddress)

		if iNode == nil {
			return
		}

		iNode.mutex.RLock()
		defer iNode.mutex.RUnlock()

		for _, iDev := range iNode.devices {
			iDev.mutex.RLock()

			if isEndpointInSlice(iDev.endpoints, source.Message.DestinationEndpoint) {
				iDev.mutex.RUnlock()

				device := internalDeviceToZDADevice(iDev)
				c(device, source.Message)
				return
			}

			iDev.mutex.RUnlock()
		}
	})
	s.zclCommunicatorCallbacks.AddCallback(match)
}

func (s *zclShim) WaitForMessage(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, i zcl.CommandIdentifier) (zcl.Message, error) {
	msgCh := make(chan zcl.Message, 1)

	match := s.zclCommunicatorCallbacks.NewMatch(func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
		return d.Identifier.IEEEAddress == address && zclMessage.SourceEndpoint == e && zclMessage.ClusterID == c && zclMessage.CommandIdentifier == i
	}, func(source communicator.MessageWithSource) {
		select {
		case msgCh <- source.Message:
		default:
		}
	})

	s.zclCommunicatorCallbacks.AddCallback(match)
	defer s.zclCommunicatorCallbacks.RemoveCallback(match)

	select {
	case msg := <-msgCh:
		return msg, nil
	case <-ctx.Done():
		return zcl.Message{}, context.DeadlineExceeded
	}
}
