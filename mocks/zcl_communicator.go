package mocks

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockZCLCommunicator struct {
	mock.Mock
}

func (m *MockZCLCommunicator) RegisterMatch(match communicator.Match) {
	m.Called(match)
}

func (m *MockZCLCommunicator) UnregisterMatch(match communicator.Match) {
	m.Called(match)
}

func (m *MockZCLCommunicator) ProcessIncomingMessage(msg zigbee.NodeIncomingMessageEvent) error {
	return m.Called(msg).Error(0)
}

func (m *MockZCLCommunicator) Request(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) error {
	return m.Called(ctx, address, requireAck, message).Error(0)
}

func (m *MockZCLCommunicator) RequestResponse(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) (zcl.Message, error) {
	args := m.Called(ctx, address, requireAck, message)
	return args.Get(0).(zcl.Message), args.Error(1)
}

func (m *MockZCLCommunicator) ReadAttributes(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributes)
	return args.Get(0).([]global.ReadAttributeResponseRecord), args.Error(1)
}

func (m *MockZCLCommunicator) WriteAttributes(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes map[zcl.AttributeID]zcl.AttributeDataTypeValue) ([]global.WriteAttributesResponseRecord, error) {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributes)
	return args.Get(0).([]global.WriteAttributesResponseRecord), args.Error(1)
}

func (m *MockZCLCommunicator) ConfigureReporting(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributeId zcl.AttributeID, dataType zcl.AttributeDataType, minimumReportingInterval uint16, maximumReportingInterval uint16, reportableChange any) error {
	return m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributeId, dataType, minimumReportingInterval, maximumReportingInterval, reportableChange).Error(0)
}

var _ communicator.Communicator = (*MockZCLCommunicator)(nil)
