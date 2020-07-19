package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
)

type addInternalCallback func(f interface{})

type deviceStore interface {
	getDevice(identifier da.Identifier) (*internalDevice, bool)
	addDevice(identifier da.Identifier, node *internalNode) *internalDevice
	removeDevice(identifier da.Identifier)
}

type nodeStore interface {
	getNode(ieeeAddress zigbee.IEEEAddress) (*internalNode, bool)
	addNode(ieeeAddress zigbee.IEEEAddress) *internalNode
	removeNode(ieeeAddress zigbee.IEEEAddress)
}

type zclCommunicatorCallbacks interface {
	NewMatch(matcher communicator.Matcher, callback func(source communicator.MessageWithSource)) communicator.Match

	AddCallback(match communicator.Match)
	RemoveCallback(match communicator.Match)
}

type zclCommunicatorRequests interface {
	Request(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) error
	RequestResponse(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) (zcl.Message, error)
}

type zclGlobalCommunicator interface {
	ReadAttributes(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error)
	ConfigureReporting(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributeId zcl.AttributeID, dataType zcl.AttributeDataType, minimumReportingInterval uint16, maximumReportingInterval uint16, reportableChange interface{}) error
}
