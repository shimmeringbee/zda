package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestZigbeeHasProductInformation_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.HasProductInformation", func(t *testing.T) {
		assert.Implements(t, (*capabilities.HasProductInformation)(nil), new(ZigbeeHasProductInformation))
	})
}

func TestZigbeeGateway_ReturnsHasProductInformationCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(capabilities.HasProductInformationFlag)
		assert.IsType(t, (*ZigbeeHasProductInformation)(nil), actualZdd)
	})
}

func TestZigbeeHasProductInformation_ProductInformation(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zhpi := zgw.capabilities[capabilities.HasProductInformationFlag].(*ZigbeeHasProductInformation)
		nonSelfDevice := da.Device{}

		_, err := zhpi.ProductInformation(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zhpi := zgw.capabilities[capabilities.HasProductInformationFlag].(*ZigbeeHasProductInformation)
		nonCapability := da.Device{Gateway: zgw}

		_, err := zhpi.ProductInformation(context.Background(), nonCapability)
		assert.Error(t, err)
	})
}

func TestZigbeeHasProductInformation_NodeEnumerationCallback(t *testing.T) {
	t.Run("queries each Device on a Node for basic product information", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zhpi := zgw.capabilities[capabilities.HasProductInformationFlag].(*ZigbeeHasProductInformation)

		ieee := zigbee.IEEEAddress(0x0102030405060708)
		iNode := zgw.addNode(ieee)

		iNode.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint:       0x01,
				ProfileID:      zigbee.ProfileHomeAutomation,
				DeviceID:       1,
				DeviceVersion:  0,
				InClusterList:  []zigbee.ClusterID{0x0000},
				OutClusterList: []zigbee.ClusterID{},
			},
			0x02: {
				Endpoint:       0x02,
				ProfileID:      zigbee.ProfileHomeAutomation,
				DeviceID:       2,
				DeviceVersion:  0,
				InClusterList:  []zigbee.ClusterID{0x0000},
				OutClusterList: []zigbee.ClusterID{},
			},
		}

		iDevOneId := iNode.findNextDeviceIdentifier()
		iDevOne := zgw.addDevice(iDevOneId, iNode)
		iDevOne.deviceID = 1
		iDevOne.endpoints = []zigbee.Endpoint{0x01}
		iDevOneManufacturer := "manu1"
		iDevOneProduct := "product1"

		iDevTwoId := iNode.findNextDeviceIdentifier()
		iDevTwo := zgw.addDevice(iDevTwoId, iNode)
		iDevTwo.deviceID = 2
		iDevTwo.endpoints = []zigbee.Endpoint{0x02}
		iDevTwoManufacturer := "manu2"
		iDevTwoProduct := "product2"

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		cr := zcl.NewCommandRegistry()
		global.Register(cr)

		expectedRequestOne := zcl.Message{
			FrameType:           zcl.FrameGlobal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           0x0000,
			SourceEndpoint:      1,
			DestinationEndpoint: 1,
			Command: &global.ReadAttributes{
				Identifier: []zcl.AttributeID{0x0004, 0x0005},
			},
		}

		appRequestOne, _ := cr.Marshal(expectedRequestOne)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieee, appRequestOne).Return(nil).Run(func(args mock.Arguments) {
			message := zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ServerToClient,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           0x0000,
				SourceEndpoint:      1,
				DestinationEndpoint: 1,
				Command: &global.ReadAttributesResponse{
					Records: []global.ReadAttributeResponseRecord{
						{
							Identifier: 0x0004,
							Status:     0,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeStringCharacter8,
								Value:    iDevOneManufacturer,
							},
						},
						{
							Identifier: 0x0005,
							Status:     0,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeStringCharacter8,
								Value:    iDevOneProduct,
							},
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

		expectedRequestTwo := zcl.Message{
			FrameType:           zcl.FrameGlobal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           0x0000,
			SourceEndpoint:      1,
			DestinationEndpoint: 2,
			Command: &global.ReadAttributes{
				Identifier: []zcl.AttributeID{0x0004, 0x0005},
			},
		}

		appRequestTwo, _ := cr.Marshal(expectedRequestTwo)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieee, appRequestTwo).Return(nil).Run(func(args mock.Arguments) {
			message := zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ServerToClient,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           0x0000,
				SourceEndpoint:      2,
				DestinationEndpoint: 1,
				Command: &global.ReadAttributesResponse{
					Records: []global.ReadAttributeResponseRecord{
						{
							Identifier: 0x0004,
							Status:     0,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeStringCharacter8,
								Value:    iDevTwoManufacturer,
							},
						},
						{
							Identifier: 0x0005,
							Status:     0,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeStringCharacter8,
								Value:    iDevTwoProduct,
							},
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

		err := zhpi.NodeEnumerationCallback(ctx, internalNodeEnumeration{node: iNode})
		assert.NoError(t, err)

		assert.Equal(t, []da.Capability{capabilities.EnumerateDeviceFlag, capabilities.LocalDebugFlag, capabilities.HasProductInformationFlag}, iDevOne.device.Capabilities)
		assert.Equal(t, []da.Capability{capabilities.EnumerateDeviceFlag, capabilities.LocalDebugFlag, capabilities.HasProductInformationFlag}, iDevTwo.device.Capabilities)

		prodInfoOne, err := zhpi.ProductInformation(ctx, iDevOne.device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoOne.Present)
		assert.Equal(t, iDevOneManufacturer, prodInfoOne.Manufacturer)
		assert.Equal(t, iDevOneProduct, prodInfoOne.Name)

		prodInfoTwo, err := zhpi.ProductInformation(ctx, iDevTwo.device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoTwo.Present)
		assert.Equal(t, iDevTwoManufacturer, prodInfoTwo.Manufacturer)
		assert.Equal(t, iDevTwoProduct, prodInfoTwo.Name)
	})

	t.Run("queries each Device on a Node for basic product information", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zhpi := zgw.capabilities[capabilities.HasProductInformationFlag].(*ZigbeeHasProductInformation)

		ieee := zigbee.IEEEAddress(0x0102030405060708)
		iNode := zgw.addNode(ieee)

		iNode.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint:       0x01,
				ProfileID:      zigbee.ProfileHomeAutomation,
				DeviceID:       1,
				DeviceVersion:  0,
				InClusterList:  []zigbee.ClusterID{0x0000},
				OutClusterList: []zigbee.ClusterID{},
			},
		}

		iDevOneId := iNode.findNextDeviceIdentifier()
		iDevOne := zgw.addDevice(iDevOneId, iNode)
		iDevOne.deviceID = 1
		iDevOne.endpoints = []zigbee.Endpoint{0x01}
		iDevOneManufacturer := "manu1"

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		cr := zcl.NewCommandRegistry()
		global.Register(cr)

		expectedRequestOne := zcl.Message{
			FrameType:           zcl.FrameGlobal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        0,
			ClusterID:           0x0000,
			SourceEndpoint:      1,
			DestinationEndpoint: 1,
			Command: &global.ReadAttributes{
				Identifier: []zcl.AttributeID{0x0004, 0x0005},
			},
		}

		appRequestOne, _ := cr.Marshal(expectedRequestOne)

		mockProvider.On("SendApplicationMessageToNode", mock.Anything, ieee, appRequestOne).Return(nil).Run(func(args mock.Arguments) {
			message := zcl.Message{
				FrameType:           zcl.FrameGlobal,
				Direction:           zcl.ServerToClient,
				TransactionSequence: 0,
				Manufacturer:        0,
				ClusterID:           0x0000,
				SourceEndpoint:      1,
				DestinationEndpoint: 1,
				Command: &global.ReadAttributesResponse{
					Records: []global.ReadAttributeResponseRecord{
						{
							Identifier: 0x0004,
							Status:     0,
							DataTypeValue: &zcl.AttributeDataTypeValue{
								DataType: zcl.TypeStringCharacter8,
								Value:    iDevOneManufacturer,
							},
						},
						{
							Identifier: 0x0005,
							Status:     1,
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

		err := zhpi.NodeEnumerationCallback(ctx, internalNodeEnumeration{node: iNode})
		assert.NoError(t, err)

		assert.Equal(t, []da.Capability{capabilities.EnumerateDeviceFlag, capabilities.LocalDebugFlag, capabilities.HasProductInformationFlag}, iDevOne.device.Capabilities)

		prodInfoOne, err := zhpi.ProductInformation(ctx, iDevOne.device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer, prodInfoOne.Present)
		assert.Equal(t, iDevOneManufacturer, prodInfoOne.Manufacturer)
		assert.Equal(t, "", prodInfoOne.Name)
	})
}
