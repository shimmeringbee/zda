package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestZigbeeOnOff_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.OnOff", func(t *testing.T) {
		assert.Implements(t, (*capabilities.OnOff)(nil), new(ZigbeeOnOff))
	})
}

func TestZigbeeGateway_ReturnsOnOffCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZOO := zgw.Capability(capabilities.OnOffFlag)
		assert.IsType(t, (*ZigbeeOnOff)(nil), actualZOO)
	})
}

func TestZigbeeOnOff_Init(t *testing.T) {
	t.Run("initialises the zigbee on off capability by registering callbacks", func(t *testing.T) {
		mIntCallbacks := mockAddInternalCallback{}
		mZclCallbacks := mockZclCommunicatorCallbacks{}

		zoo := ZigbeeOnOff{
			addInternalCallback:      mIntCallbacks.addInternalCallback,
			zclCommunicatorCallbacks: &mZclCallbacks,
		}

		mIntCallbacks.On("addInternalCallback", mock.Anything).Once()

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
		ieeeAddress := zigbee.IEEEAddress(0x01)
		deviceEndpoint := zigbee.Endpoint(0x05)

		deviceId := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieeeAddress,
			SubIdentifier: 0,
		}

		device := &internalDevice{
			device: da.Device{
				Identifier:   deviceId,
				Capabilities: []da.Capability{},
			},
			endpoints: []zigbee.Endpoint{deviceEndpoint},
			mutex:     &sync.RWMutex{},
		}

		node := &internalNode{
			ieeeAddress: ieeeAddress,
			mutex:       &sync.RWMutex{},
			devices:     map[IEEEAddressWithSubIdentifier]*internalDevice{deviceId: device},
			nodeDesc:    zigbee.NodeDescription{},
			endpoints:   []zigbee.Endpoint{deviceEndpoint},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{
				deviceEndpoint: {
					Endpoint:      deviceEndpoint,
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
			transactionSequences: make(chan uint8, 1),
			supportsAPSAck:       false,
		}

		expectedTransactionSeq := uint8(0x01)
		node.transactionSequences <- expectedTransactionSeq

		mockNodeBinder := mockNodeBinder{}
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		zoo := ZigbeeOnOff{
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
			nodeBinder:            &mockNodeBinder,
		}

		mockNodeBinder.On("BindNodeToController", mock.Anything, ieeeAddress, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, zcl.OnOffId).Return(nil)
		mockZclGlobalCommunicator.On("ConfigureReporting", mock.Anything, ieeeAddress, false, zcl.OnOffId, zigbee.NoManufacturer, deviceEndpoint, DefaultGatewayHomeAutomationEndpoint, expectedTransactionSeq, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		err := zoo.NodeEnumerationCallback(context.Background(), internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		has := device.device.HasCapability(capabilities.OnOffFlag)
		assert.True(t, has)

		mockNodeBinder.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}

func TestZigbeeOnOff_On(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonSelfDevice := da.Device{}

		err := zoo.On(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonCapability := da.Device{Gateway: zgw}

		err := zoo.On(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("sends On command to endpoint on device", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieeeAddress := zigbee.IEEEAddress(0x01)

		iNode := zgw.addNode(ieeeAddress)
		iDev := zgw.addDevice(iNode.nextDeviceIdentifier(), iNode)

		iDev.device.Capabilities = append(iDev.device.Capabilities, capabilities.OnOffFlag)
		iNode.supportsAPSAck = true

		iNode.endpointDescriptions[zigbee.Endpoint(0x01)] = zigbee.EndpointDescription{
			Endpoint:      0x01,
			InClusterList: []zigbee.ClusterID{zcl.OnOffId},
		}

		iDev.endpoints = []zigbee.Endpoint{0x01}

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)

		expectedRequest := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      1,
			DestinationEndpoint: 0x01,
			Command:             &onoff.On{},
		}

		cr := zcl.NewCommandRegistry()
		global.Register(cr)
		onoff.Register(cr)

		request, _ := cr.Marshal(expectedRequest)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieeeAddress, request, true).Return(nil)

		err := zoo.On(context.Background(), iDev.device)
		assert.NoError(t, err)
	})
}

func TestZigbeeOnOff_Off(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonSelfDevice := da.Device{}

		err := zoo.Off(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonCapability := da.Device{Gateway: zgw}

		err := zoo.Off(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("sends Off command to endpoint on device", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieeeAddress := zigbee.IEEEAddress(0x01)

		iNode := zgw.addNode(ieeeAddress)
		iDev := zgw.addDevice(iNode.nextDeviceIdentifier(), iNode)

		iDev.device.Capabilities = append(iDev.device.Capabilities, capabilities.OnOffFlag)
		iNode.supportsAPSAck = true

		iNode.endpointDescriptions[zigbee.Endpoint(0x01)] = zigbee.EndpointDescription{
			Endpoint:      0x01,
			InClusterList: []zigbee.ClusterID{zcl.OnOffId},
		}

		iDev.endpoints = []zigbee.Endpoint{0x01}

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)

		expectedRequest := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      1,
			DestinationEndpoint: 0x01,
			Command:             &onoff.Off{},
		}

		cr := zcl.NewCommandRegistry()
		global.Register(cr)
		onoff.Register(cr)

		request, _ := cr.Marshal(expectedRequest)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieeeAddress, request, true).Return(nil)

		err := zoo.Off(context.Background(), iDev.device)
		assert.NoError(t, err)
	})
}

func TestZigbeeOnOff_State(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonSelfDevice := da.Device{}

		_, err := zoo.State(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)
		nonCapability := da.Device{Gateway: zgw}

		_, err := zoo.State(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("state is set to true if attribute has been reported", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieeeAddress := zigbee.IEEEAddress(0x01)

		iNode := zgw.addNode(ieeeAddress)
		iDev := zgw.addDevice(iNode.nextDeviceIdentifier(), iNode)

		iDev.device.Capabilities = append(iDev.device.Capabilities, capabilities.OnOffFlag)
		iNode.supportsAPSAck = true

		iNode.endpointDescriptions[zigbee.Endpoint(0x01)] = zigbee.EndpointDescription{
			Endpoint:      0x01,
			InClusterList: []zigbee.ClusterID{zcl.OnOffId},
		}

		iDev.endpoints = []zigbee.Endpoint{0x01}

		report := zcl.Message{
			FrameType:           zcl.FrameGlobal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      1,
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
		}

		zoo := zgw.capabilities[capabilities.OnOffFlag].(*ZigbeeOnOff)

		cr := zcl.NewCommandRegistry()
		global.Register(cr)
		onoff.Register(cr)

		request, _ := cr.Marshal(report)

		zgw.communicator.ProcessIncomingMessage(zigbee.NodeIncomingMessageEvent{
			Node: zigbee.Node{
				IEEEAddress: ieeeAddress,
			},
			IncomingMessage: zigbee.IncomingMessage{
				GroupID:              0,
				SourceIEEEAddress:    0,
				SourceNetworkAddress: 0,
				Broadcast:            false,
				Secure:               false,
				LinkQuality:          0,
				Sequence:             0,
				ApplicationMessage:   request,
			},
		})

		time.Sleep(10 * time.Millisecond)

		value, err := zoo.State(context.Background(), iDev.device)
		assert.NoError(t, err)
		assert.True(t, value)
	})
}
