package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestZigbeeOnOff_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.HasProductInformation", func(t *testing.T) {
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

func TestZigbeeOnOff_NodeEnumerationCallback(t *testing.T) {
	t.Run("adds Onoff capability to device with OnOff cluster", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		cr := zcl.NewCommandRegistry()
		global.Register(cr)

		oo := zgw.Capability(capabilities.OnOffFlag)
		zoo := oo.(*ZigbeeOnOff)

		ieee := zigbee.IEEEAddress(1)

		node := zgw.addNode(ieee)
		dev := zgw.addDevice(node.nextDeviceIdentifier(), node)

		mockProvider.On("BindNodeToController", mock.Anything, ieee, zigbee.Endpoint(1), zigbee.Endpoint(1), zcl.OnOffId).Return(nil)

		node.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint:       0x01,
				ProfileID:      0,
				DeviceID:       0,
				DeviceVersion:  0,
				InClusterList:  []zigbee.ClusterID{zcl.OnOffId},
				OutClusterList: nil,
			},
		}

		dev.endpoints = []zigbee.Endpoint{0x01}

		expectedRequestOne := zcl.Message{
			FrameType:           zcl.FrameGlobal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           zcl.OnOffId,
			SourceEndpoint:      1,
			DestinationEndpoint: 1,
			Command: &global.ConfigureReporting{
				Records: []global.ConfigureReportingRecord{
					{
						Direction:        0,
						Identifier:       onoff.OnOff,
						DataType:         zcl.TypeBoolean,
						MinimumInterval:  0,
						MaximumInterval:  60,
						ReportableChange: &zcl.AttributeDataValue{},
						Timeout:          0,
					},
				},
			},
		}

		appRequestOne, _ := cr.Marshal(expectedRequestOne)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieee, appRequestOne, false).Return(nil).Run(func(args mock.Arguments) {
			message := zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ServerToClient,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           zcl.OnOffId,
				SourceEndpoint:      1,
				DestinationEndpoint: 1,
				Command: &global.ConfigureReportingResponse{
					Records: []global.ConfigureReportingResponseRecord{
						{
							Status:     0,
							Direction:  0,
							Identifier: onoff.OnOff,
						},
					},
				},
			}

			appMessageReply, _ := cr.Marshal(message)

			zgw.communicator.ProcessIncomingMessage(zigbee.NodeIncomingMessageEvent{
				Node: zigbee.Node{
					IEEEAddress:    ieee,
					NetworkAddress: 0,
					LogicalType:    0,
					LQI:            0,
					Depth:          0,
					LastDiscovered: time.Time{},
					LastReceived:   time.Time{},
				},
				IncomingMessage: zigbee.IncomingMessage{
					GroupID:              0,
					SourceIEEEAddress:    ieee,
					SourceNetworkAddress: 0,
					Broadcast:            false,
					Secure:               false,
					LinkQuality:          0,
					Sequence:             0,
					ApplicationMessage:   appMessageReply,
				},
			})
		})

		err := zoo.NodeEnumerationCallback(context.Background(), internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		has := dev.device.HasCapability(capabilities.OnOffFlag)
		assert.True(t, has)

		mockProvider.AssertExpectations(t)
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

		value, err := zoo.State(context.Background(), iDev.device)
		assert.NoError(t, err)
		assert.True(t, value)
	})
}
