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
)

func TestZigbeeHasProductInformation_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.HasProductInformation", func(t *testing.T) {
		assert.Implements(t, (*capabilities.HasProductInformation)(nil), new(ZigbeeHasProductInformation))
	})
}

func TestZigbeeHasProductInformation_ProductInformation(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zhpi := ZigbeeHasProductInformation{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		_, err := zhpi.ProductInformation(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zhpi := ZigbeeHasProductInformation{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zhpi.gateway}

		_, err := zhpi.ProductInformation(context.Background(), nonCapability)
		assert.Error(t, err)
	})
}

func TestZigbeeHasProductInformation_NodeEnumerationCallback(t *testing.T) {
	t.Run("queries each Device on a Node for basic product information", func(t *testing.T) {
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}

		nt, node, devices := generateNodeTableWithData(2)

		for _, endpoint := range node.endpoints {
			endpointDescription := node.endpointDescriptions[endpoint]
			endpointDescription.InClusterList = []zigbee.ClusterID{zcl.BasicId}
			node.endpointDescriptions[endpoint] = endpointDescription
		}

		zhpi := ZigbeeHasProductInformation{
			gateway:               &mockGateway{},
			nodeTable:             nt,
			internalCallbacks:     nil,
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
		}

		manufactureres := []string{"manu1", "manu2"}
		products := []string{"product1", "product2"}

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), mock.Anything, []zcl.AttributeID{0x0004, 0x0005}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: 0x0004,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    manufactureres[0],
					},
				},
				{
					Identifier: 0x0005,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    products[0],
					},
				},
			}, nil)

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), mock.Anything, []zcl.AttributeID{0x0004, 0x0005}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: 0x0004,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    manufactureres[1],
					},
				},
				{
					Identifier: 0x0005,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    products[1],
					},
				},
			}, nil)

		ctx := context.Background()

		err := zhpi.NodeEnumerationCallback(ctx, internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		assert.Contains(t, devices[0].capabilities, capabilities.HasProductInformationFlag)
		assert.Contains(t, devices[1].capabilities, capabilities.HasProductInformationFlag)

		prodInfoOne, err := zhpi.ProductInformation(ctx, devices[0].toDevice(zhpi.gateway))
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoOne.Present)
		assert.Equal(t, manufactureres[0], prodInfoOne.Manufacturer)
		assert.Equal(t, products[0], prodInfoOne.Name)

		prodInfoTwo, err := zhpi.ProductInformation(ctx, devices[1].toDevice(zhpi.gateway))
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoTwo.Present)
		assert.Equal(t, manufactureres[1], prodInfoTwo.Manufacturer)
		assert.Equal(t, products[1], prodInfoTwo.Name)

		mockZclGlobalCommunicator.AssertExpectations(t)
	})

	t.Run("handles responses with unsupported attributes", func(t *testing.T) {
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}
		nt, node, devices := generateNodeTableWithData(2)

		zhpi := ZigbeeHasProductInformation{
			gateway:               &mockGateway{},
			nodeTable:             nt,
			internalCallbacks:     nil,
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
		}

		for _, endpoint := range node.endpoints {
			endpointDescription := node.endpointDescriptions[endpoint]
			endpointDescription.InClusterList = []zigbee.ClusterID{zcl.BasicId}
			node.endpointDescriptions[endpoint] = endpointDescription
		}

		manufacturers := []string{"manu1", "manu2"}
		products := []string{"product1", "product2"}

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), mock.Anything, []zcl.AttributeID{0x0004, 0x0005}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: 0x0004,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    manufacturers[0],
					},
				},
				{
					Identifier:    0x0005,
					Status:        1,
					DataTypeValue: nil,
				},
			}, nil)

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), mock.Anything, []zcl.AttributeID{0x0004, 0x0005}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier:    0x0004,
					Status:        1,
					DataTypeValue: nil,
				},
				{
					Identifier: 0x0005,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    products[1],
					},
				},
			}, nil)

		ctx := context.Background()

		err := zhpi.NodeEnumerationCallback(ctx, internalNodeEnumeration{node: node})
		assert.NoError(t, err)

		assert.Contains(t, devices[0].capabilities, capabilities.HasProductInformationFlag)
		assert.Contains(t, devices[1].capabilities, capabilities.HasProductInformationFlag)

		prodInfoOne, err := zhpi.ProductInformation(ctx, devices[0].toDevice(zhpi.gateway))
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer, prodInfoOne.Present)
		assert.Equal(t, manufacturers[0], prodInfoOne.Manufacturer)

		prodInfoTwo, err := zhpi.ProductInformation(ctx, devices[1].toDevice(zhpi.gateway))
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Name, prodInfoTwo.Present)
		assert.Equal(t, products[1], prodInfoTwo.Name)

		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}
