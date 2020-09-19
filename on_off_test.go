package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math"
	"sync"
	"testing"
	"time"
)

func TestZigbeeOnOff_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.OnOff", func(t *testing.T) {
		assert.Implements(t, (*capabilities.OnOff)(nil), new(ZigbeeOnOff))
	})
}

func TestZigbeeOnOff_Init(t *testing.T) {
	t.Run("initialises the zigbee on off capability by registering internalCallbacks", func(t *testing.T) {
		mIntCallbacks := mockAdderCaller{}
		mZclCallbacks := mockZclCommunicatorCallbacks{}

		zoo := ZigbeeOnOff{
			internalCallbacks:        &mIntCallbacks,
			zclCommunicatorCallbacks: &mZclCallbacks,
		}

		mIntCallbacks.On("Add", mock.Anything).Twice()

		returnedMatch := communicator.Match{
			Id:       1,
			Matcher:  nil,
			Callback: nil,
		}
		mZclCallbacks.On("NewMatch", mock.Anything, mock.Anything).Return(returnedMatch).Once()
		mZclCallbacks.On("AddCallback", returnedMatch).Once()

		zoo.Init()

		mIntCallbacks.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_NodeEnumerationCallback(t *testing.T) {
	t.Run("adds Onoff capability to device with OnOff cluster, attempts to bind and configure reporting", func(t *testing.T) {
		mockNodeBinder := mockNodeBinder{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
			nodeBinder:            &mockNodeBinder,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		expectedTransactionSeq := uint8(1)

		mockNodeBinder.On("BindNodeToController", mock.Anything, node.ieeeAddress, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId).Return(nil)
		mockZclGlobalCommunicator.On("ConfigureReporting", mock.Anything, node.ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, expectedTransactionSeq, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		err := zoo.NodeEnumerationCallback(context.Background(), internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		has := device.toDevice().HasCapability(capabilities.OnOffFlag)
		assert.True(t, has)

		mockNodeBinder.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})

	t.Run("the device is set to require polling if binding fails", func(t *testing.T) {
		mockNodeBinder := mockNodeBinder{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
			nodeBinder:            &mockNodeBinder,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockNodeBinder.On("BindNodeToController", mock.Anything, node.ieeeAddress, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId).Return(errors.New("failure")).Times(DefaultNetworkRetries)
		mockZclGlobalCommunicator.On("ConfigureReporting", mock.Anything, node.ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, mock.Anything, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		err := zoo.NodeEnumerationCallback(context.Background(), internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		has := device.toDevice().HasCapability(capabilities.OnOffFlag)
		assert.True(t, has)

		assert.True(t, device.onOffState.requiresPolling)

		mockNodeBinder.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})

	t.Run("the device is set to require polling if configure reporting fails", func(t *testing.T) {
		mockNodeBinder := mockNodeBinder{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
			nodeBinder:            &mockNodeBinder,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockNodeBinder.On("BindNodeToController", mock.Anything, node.ieeeAddress, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId).Return(nil)
		mockZclGlobalCommunicator.On("ConfigureReporting", mock.Anything, node.ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, mock.Anything, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(errors.New("failure")).Times(DefaultNetworkRetries)

		err := zoo.NodeEnumerationCallback(context.Background(), internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		has := device.toDevice().HasCapability(capabilities.OnOffFlag)
		assert.True(t, has)

		assert.True(t, device.onOffState.requiresPolling)

		mockNodeBinder.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_On(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		err := zoo.On(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zoo.gateway}

		err := zoo.On(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("sends On command to endpoint on device", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}
		mockZclCommunicatorRequests := mockZclCommunicatorRequests{}

		zoo := ZigbeeOnOff{
			gateway:                 &mockGateway{},
			deviceStore:             &mockDeviceStore,
			zclCommunicatorRequests: &mockZclCommunicatorRequests,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway
		device.capabilities = []da.Capability{capabilities.OnOffFlag}

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockDeviceStore.On("getDevice", device.generateIdentifier()).Return(device, true)

		expectedRequest := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 1,
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
			DestinationEndpoint: deviceEndpoint,
			Command:             &onoff.On{},
		}
		mockZclCommunicatorRequests.On("Request", mock.Anything, node.ieeeAddress, false, expectedRequest).Return(nil)

		err := zoo.On(context.Background(), device.toDevice())
		assert.NoError(t, err)

		mockDeviceStore.AssertExpectations(t)
		mockZclCommunicatorRequests.AssertExpectations(t)
	})

	t.Run("polls state after sending an On command to endpoint on device which requires polling", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}
		mockZclCommunicatorRequests := mockZclCommunicatorRequests{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			gateway:                 &mockGateway{},
			deviceStore:             &mockDeviceStore,
			zclCommunicatorRequests: &mockZclCommunicatorRequests,
			zclGlobalCommunicator:   &mockZclGlobalCommunicator,
		}

		node, device := generateTestNodeAndDevice()
		node.nodeDesc.LogicalType = zigbee.Router
		node.gateway = zoo.gateway
		device.capabilities = []da.Capability{capabilities.OnOffFlag}
		device.onOffState.requiresPolling = true

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockDeviceStore.On("getDevice", device.generateIdentifier()).Return(device, true)

		mockZclCommunicatorRequests.On("Request", mock.Anything, node.ieeeAddress, false, mock.Anything).Return(nil)
		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), mock.Anything, []zcl.AttributeID{onoff.OnOff}).Return([]global.ReadAttributeResponseRecord{}, errors.New("unimplemented"))

		err := zoo.On(context.Background(), device.toDevice())
		assert.NoError(t, err)

		time.Sleep(time.Duration(1.5 * float64(delayAfterSetForPolling)))

		mockDeviceStore.AssertExpectations(t)
		mockZclCommunicatorRequests.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_Off(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		err := zoo.Off(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zoo.gateway}

		err := zoo.Off(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("sends Off command to endpoint on device", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}
		mockZclCommunicatorRequests := mockZclCommunicatorRequests{}

		zoo := ZigbeeOnOff{
			gateway:                 &mockGateway{},
			deviceStore:             &mockDeviceStore,
			zclCommunicatorRequests: &mockZclCommunicatorRequests,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway
		device.capabilities = []da.Capability{capabilities.OnOffFlag}

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockDeviceStore.On("getDevice", device.generateIdentifier()).Return(device, true)

		expectedRequest := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 1,
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
			DestinationEndpoint: deviceEndpoint,
			Command:             &onoff.Off{},
		}
		mockZclCommunicatorRequests.On("Request", mock.Anything, node.ieeeAddress, false, expectedRequest).Return(nil)

		err := zoo.Off(context.Background(), device.toDevice())
		assert.NoError(t, err)

		mockDeviceStore.AssertExpectations(t)
		mockZclCommunicatorRequests.AssertExpectations(t)
	})

	t.Run("polls state after sending an Off command to endpoint on device which requires polling", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}
		mockZclCommunicatorRequests := mockZclCommunicatorRequests{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			gateway:                 &mockGateway{},
			deviceStore:             &mockDeviceStore,
			zclCommunicatorRequests: &mockZclCommunicatorRequests,
			zclGlobalCommunicator:   &mockZclGlobalCommunicator,
		}

		node, device := generateTestNodeAndDevice()
		node.nodeDesc.LogicalType = zigbee.Router
		node.gateway = zoo.gateway
		device.capabilities = []da.Capability{capabilities.OnOffFlag}
		device.onOffState.requiresPolling = true

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockDeviceStore.On("getDevice", device.generateIdentifier()).Return(device, true)

		mockZclCommunicatorRequests.On("Request", mock.Anything, node.ieeeAddress, false, mock.Anything).Return(nil)
		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), mock.Anything, []zcl.AttributeID{onoff.OnOff}).Return([]global.ReadAttributeResponseRecord{}, errors.New("unimplemented"))

		err := zoo.Off(context.Background(), device.toDevice())
		assert.NoError(t, err)

		time.Sleep(time.Duration(1.5 * float64(delayAfterSetForPolling)))

		mockDeviceStore.AssertExpectations(t)
		mockZclCommunicatorRequests.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_State(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		_, err := zoo.State(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zoo := ZigbeeOnOff{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zoo.gateway}

		_, err := zoo.State(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("state is set to true if attribute has been reported", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}
		mockNodeStore := mockNodeStore{}
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything)

		zoo := ZigbeeOnOff{
			gateway:     &mockGateway{},
			nodeStore:   &mockNodeStore,
			deviceStore: &mockDeviceStore,
			eventSender: &mockEventSender,
		}

		node, device := generateTestNodeAndDevice()
		node.gateway = zoo.gateway
		device.capabilities = []da.Capability{capabilities.OnOffFlag}

		deviceEndpoint := node.endpoints[0]
		endpointDescription := node.endpointDescriptions[deviceEndpoint]
		endpointDescription.InClusterList = []zigbee.ClusterID{zcl.OnOffId}
		node.endpointDescriptions[deviceEndpoint] = endpointDescription

		mockNodeStore.On("getNode", node.ieeeAddress).Return(node, true)
		mockDeviceStore.On("getDevice", device.generateIdentifier()).Return(device, true)

		zoo.incomingReportAttributes(communicator.MessageWithSource{
			SourceAddress: node.ieeeAddress,
			Message: zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ClientToServer,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           zcl.OnOffId,
				SourceEndpoint:      deviceEndpoint,
				DestinationEndpoint: DefaultGatewayHomeAutomationEndpoint,
				Command: &global.ReportAttributes{
					Records: []global.ReportAttributesRecord{
						{
							Identifier: onoff.OnOff,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeBoolean,
								Value:    true,
							},
						},
					},
				},
			},
		})

		value, err := zoo.State(context.Background(), device.toDevice())
		assert.NoError(t, err)
		assert.True(t, value)

		mockDeviceStore.AssertExpectations(t)
		mockNodeStore.AssertExpectations(t)
	})
}

func generateTestNodeAndDevice() (*internalNode, *internalDevice) {
	node, devices := generateTestNodeAndDevices(1)
	return node, devices[0]
}

func generateTestNodeAndDevices(deviceCount uint8) (*internalNode, []*internalDevice) {
	ieeeAddress := zigbee.GenerateLocalAdministeredIEEEAddress()

	var retDevices []*internalDevice

	node := &internalNode{
		ieeeAddress:          ieeeAddress,
		mutex:                &sync.RWMutex{},
		devices:              map[uint8]*internalDevice{},
		nodeDesc:             zigbee.NodeDescription{},
		endpoints:            []zigbee.Endpoint{},
		endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		transactionSequences: make(chan uint8, math.MaxUint8),
		supportsAPSAck:       false,
	}

	for i := uint8(1); i < math.MaxUint8; i++ {
		node.transactionSequences <- i
	}

	for subId := uint8(0); subId < deviceCount; subId++ {
		deviceId := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieeeAddress,
			SubIdentifier: subId,
		}

		deviceEndpoint := zigbee.Endpoint(subId)

		device := &internalDevice{
			subidentifier: deviceId.SubIdentifier,
			capabilities:  []da.Capability{},
			endpoints:     []zigbee.Endpoint{deviceEndpoint},
			mutex:         &sync.RWMutex{},
			node:          node,
		}

		node.devices[deviceId.SubIdentifier] = device
		node.endpoints = append(node.endpoints, deviceEndpoint)
		node.endpointDescriptions[deviceEndpoint] = zigbee.EndpointDescription{
			Endpoint:       deviceEndpoint,
			ProfileID:      0,
			DeviceID:       uint16(subId),
			DeviceVersion:  uint8(1),
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		retDevices = append(retDevices, device)
	}

	return node, retDevices
}

func TestZigbeeOnOff_NodeJoinCallback(t *testing.T) {
	t.Run("registers new nodes with the poller and callback function when they join", func(t *testing.T) {
		node := &internalNode{}
		mockPoller := mockPoller{}

		zoo := ZigbeeOnOff{
			poller: &mockPoller,
		}

		mockPoller.On("AddNode", node, pollInterval, mock.AnythingOfType("func(context.Context, *zda.internalNode)"))

		err := zoo.NodeJoinCallback(context.Background(), internalNodeJoin{node: node})
		assert.NoError(t, err)

		mockPoller.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_pollNode(t *testing.T) {
	t.Run("queries the OnOff state of each device marked with requiresPolling", func(t *testing.T) {
		node, device := generateTestNodeAndDevice()
		node.nodeDesc.LogicalType = zigbee.Router
		device.onOffState.requiresPolling = true
		device.capabilities = []da.Capability{capabilities.OnOffFlag}
		node.endpointDescriptions[0] = zigbee.EndpointDescription{
			InClusterList: []zigbee.ClusterID{zcl.OnOffId},
		}

		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything)

		zoo := ZigbeeOnOff{
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
			eventSender:           &mockEventSender,
		}

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.OnOffId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, device.endpoints[0], uint8(1), []zcl.AttributeID{onoff.OnOff}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: onoff.OnOff,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeBoolean,
						Value:    true,
					},
				},
			}, nil)

		ctx, done := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer done()

		zoo.pollNode(ctx, node)

		assert.True(t, device.onOffState.State)

		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_setState(t *testing.T) {
	t.Run("setting a new state issues a state change event to the gateway consumer", func(t *testing.T) {
		_, device := generateTestNodeAndDevice()
		mockEventSender := mockEventSender{}

		zoo := ZigbeeOnOff{
			eventSender: &mockEventSender,
		}

		expectedEvent := capabilities.OnOffState{Device: device.toDevice(), State: true}

		mockEventSender.On("sendEvent", expectedEvent)

		zoo.setState(device, true)

		mockEventSender.AssertExpectations(t)
	})
}
