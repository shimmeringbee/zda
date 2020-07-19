package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type addInternalCallback func(f interface{})

type deviceStore interface {
	getDevice(identifier da.Identifier) (*internalDevice, bool)
	addDevice(identifier da.Identifier, node *internalNode) *internalDevice
	removeDevice(identifier da.Identifier)
}

type mockDeviceStore struct {
	mock.Mock
}

func (m *mockDeviceStore) getDevice(identifier da.Identifier) (*internalDevice, bool) {
	args := m.Called(identifier)
	return args.Get(0).(*internalDevice), args.Bool(1)
}

func (m *mockDeviceStore) addDevice(identifier da.Identifier, node *internalNode) *internalDevice {
	args := m.Called(identifier, node)
	return args.Get(0).(*internalDevice)
}

func (m *mockDeviceStore) removeDevice(identifier da.Identifier) {
	m.Called(identifier)
}

type nodeStore interface {
	getNode(ieeeAddress zigbee.IEEEAddress) (*internalNode, bool)
	addNode(ieeeAddress zigbee.IEEEAddress) *internalNode
	removeNode(ieeeAddress zigbee.IEEEAddress)
}

type mockNodeStore struct {
	mock.Mock
}

func (m *mockNodeStore) getNode(ieeeAddress zigbee.IEEEAddress) (*internalNode, bool) {
	args := m.Called(ieeeAddress)
	return args.Get(0).(*internalNode), args.Bool(1)
}

func (m *mockNodeStore) addNode(ieeeAddress zigbee.IEEEAddress) *internalNode {
	args := m.Called(ieeeAddress)
	return args.Get(0).(*internalNode)
}

func (m *mockNodeStore) removeNode(ieeeAddress zigbee.IEEEAddress) {
	m.Called(ieeeAddress)
}

type zclCommunicatorCallbacks interface {
	NewMatch(matcher communicator.Matcher, callback func(source communicator.MessageWithSource)) communicator.Match

	AddCallback(match communicator.Match)
	RemoveCallback(match communicator.Match)
}

type mockZclCommunicatorCallbacks struct {
	mock.Mock
}

func (m *mockZclCommunicatorCallbacks) NewMatch(matcher communicator.Matcher, callback func(source communicator.MessageWithSource)) communicator.Match {
	args := m.Called(matcher, callback)
	return args.Get(0).(communicator.Match)
}

func (m *mockZclCommunicatorCallbacks) AddCallback(match communicator.Match) {
	m.Called(match)
}

func (m *mockZclCommunicatorCallbacks) RemoveCallback(match communicator.Match) {
	m.Called(match)
}

type zclCommunicatorRequests interface {
	Request(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) error
	RequestResponse(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) (zcl.Message, error)
}

type mockZclCommunicatorRequests struct {
	mock.Mock
}

func (m *mockZclCommunicatorRequests) Request(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) error {
	args := m.Called(ctx, address, requireAck, message)
	return args.Error(0)
}

func (m *mockZclCommunicatorRequests) RequestResponse(ctx context.Context, address zigbee.IEEEAddress, requireAck bool, message zcl.Message) (zcl.Message, error) {
	args := m.Called(ctx, address, requireAck, message)
	return args.Get(1).(zcl.Message), args.Error(1)
}

type zclGlobalCommunicator interface {
	ReadAttributes(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error)
	ConfigureReporting(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributeId zcl.AttributeID, dataType zcl.AttributeDataType, minimumReportingInterval uint16, maximumReportingInterval uint16, reportableChange interface{}) error
}

type mockZclGlobalCommunicator struct {
	mock.Mock
}

func (m *mockZclGlobalCommunicator) ReadAttributes(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributes)
	return args.Get(0).([]global.ReadAttributeResponseRecord), args.Error(1)
}

func (m *mockZclGlobalCommunicator) ConfigureReporting(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributeId zcl.AttributeID, dataType zcl.AttributeDataType, minimumReportingInterval uint16, maximumReportingInterval uint16, reportableChange interface{}) error {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributeId, dataType, minimumReportingInterval, maximumReportingInterval, reportableChange)
	return args.Error(0)
}
