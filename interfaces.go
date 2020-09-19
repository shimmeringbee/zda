package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
	"time"
)

type mockAdderCaller struct {
	mock.Mock
}

func (m *mockAdderCaller) Add(f interface{}) {
	m.Called(f)
}

func (m *mockAdderCaller) Call(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type deviceStore interface {
	getDevice(identifier IEEEAddressWithSubIdentifier) (*internalDevice, bool)
	addDevice(identifier IEEEAddressWithSubIdentifier, node *internalNode) *internalDevice
	removeDevice(identifier IEEEAddressWithSubIdentifier)
}

type mockDeviceStore struct {
	mock.Mock
}

func (m *mockDeviceStore) getDevice(identifier IEEEAddressWithSubIdentifier) (*internalDevice, bool) {
	args := m.Called(identifier)
	return args.Get(0).(*internalDevice), args.Bool(1)
}

func (m *mockDeviceStore) addDevice(identifier IEEEAddressWithSubIdentifier, node *internalNode) *internalDevice {
	args := m.Called(identifier, node)
	return args.Get(0).(*internalDevice)
}

func (m *mockDeviceStore) removeDevice(identifier IEEEAddressWithSubIdentifier) {
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

type mockNodeBinder struct {
	mock.Mock
}

func (m *mockNodeBinder) BindNodeToController(ctx context.Context, nodeAddress zigbee.IEEEAddress, sourceEndpoint zigbee.Endpoint, destinationEndpoint zigbee.Endpoint, cluster zigbee.ClusterID) error {
	args := m.Called(ctx, nodeAddress, sourceEndpoint, destinationEndpoint, cluster)
	return args.Error(0)
}

func (m *mockNodeBinder) UnbindNodeFromController(ctx context.Context, nodeAddress zigbee.IEEEAddress, sourceEndpoint zigbee.Endpoint, destinationEndpoint zigbee.Endpoint, cluster zigbee.ClusterID) error {
	args := m.Called(ctx, nodeAddress, sourceEndpoint, destinationEndpoint, cluster)
	return args.Error(0)
}

type mockGateway struct {
	mock.Mock
}

func (m *mockGateway) ReadEvent(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *mockGateway) Capability(capability da.Capability) interface{} {
	args := m.Called(capability)
	return args.Get(0)
}

func (m *mockGateway) Self() da.Device {
	args := m.Called()
	return args.Get(0).(da.Device)
}

func (m *mockGateway) Devices() []da.Device {
	args := m.Called()
	return args.Get(0).([]da.Device)
}

func (m *mockGateway) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockGateway) Stop() error {
	args := m.Called()
	return args.Error(0)
}

type poller interface {
	AddNode(*internalNode, time.Duration, func(context.Context, *internalNode))
}

type mockPoller struct {
	mock.Mock
}

func (m *mockPoller) AddNode(node *internalNode, interval time.Duration, fn func(context.Context, *internalNode)) {
	m.Called(node, interval, fn)
}

type eventSender interface {
	sendEvent(event interface{})
}

type mockEventSender struct {
	mock.Mock
}

func (m *mockEventSender) sendEvent(event interface{}) {
	m.Called(event)
}

type mockNetworkJoining struct {
	mock.Mock
}

func (m *mockNetworkJoining) PermitJoin(ctx context.Context, allRouters bool) error {
	args := m.Called(ctx, allRouters)
	return args.Error(0)
}

func (m *mockNetworkJoining) DenyJoin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockNodeQuerier struct {
	mock.Mock
}

func (m *mockNodeQuerier) QueryNodeDescription(ctx context.Context, networkAddress zigbee.IEEEAddress) (zigbee.NodeDescription, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).(zigbee.NodeDescription), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpoints(ctx context.Context, networkAddress zigbee.IEEEAddress) ([]zigbee.Endpoint, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).([]zigbee.Endpoint), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpointDescription(ctx context.Context, networkAddress zigbee.IEEEAddress, endpoint zigbee.Endpoint) (zigbee.EndpointDescription, error) {
	args := m.Called(ctx, networkAddress, endpoint)
	return args.Get(0).(zigbee.EndpointDescription), args.Error(1)
}
