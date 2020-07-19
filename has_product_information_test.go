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
		zhpi := ZigbeeHasProductInformation{
			Gateway: &mockGateway{},
		}

		nonSelfDevice := da.Device{}

		_, err := zhpi.ProductInformation(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zhpi := ZigbeeHasProductInformation{
			Gateway: &mockGateway{},
		}

		nonCapability := da.Device{Gateway: zhpi.Gateway}

		_, err := zhpi.ProductInformation(context.Background(), nonCapability)
		assert.Error(t, err)
	})
}

func TestZigbeeHasProductInformation_NodeEnumerationCallback(t *testing.T) {
	t.Run("queries each Device on a Node for basic product information", func(t *testing.T) {
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}
		mockDeviceStore := mockDeviceStore{}

		zhpi := ZigbeeHasProductInformation{
			Gateway:               &mockGateway{},
			deviceStore:           &mockDeviceStore,
			addInternalCallback:   nil,
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
		}

		node, devices := generateTestNodeAndDevices(2)
		for i := 0; i < len(devices); i++ {
			devices[i].device.Gateway = zhpi.Gateway
		}

		for _, endpoint := range node.endpoints {
			endpointDescription := node.endpointDescriptions[endpoint]
			endpointDescription.InClusterList = []zigbee.ClusterID{zcl.BasicId}
			node.endpointDescriptions[endpoint] = endpointDescription
		}

		mockDeviceStore.On("getDevice", devices[0].device.Identifier).Return(devices[0], true)
		mockDeviceStore.On("getDevice", devices[1].device.Identifier).Return(devices[1], true)

		manufactureres := []string{"manu1", "manu2"}
		products := []string{"product1", "product2"}

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), uint8(1), []zcl.AttributeID{0x0004, 0x0005}).
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

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), uint8(2), []zcl.AttributeID{0x0004, 0x0005}).
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

		assert.Equal(t, []da.Capability{capabilities.HasProductInformationFlag}, devices[0].device.Capabilities)
		assert.Equal(t, []da.Capability{capabilities.HasProductInformationFlag}, devices[1].device.Capabilities)

		prodInfoOne, err := zhpi.ProductInformation(ctx, devices[0].device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoOne.Present)
		assert.Equal(t, manufactureres[0], prodInfoOne.Manufacturer)
		assert.Equal(t, products[0], prodInfoOne.Name)

		prodInfoTwo, err := zhpi.ProductInformation(ctx, devices[1].device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer+capabilities.Name, prodInfoTwo.Present)
		assert.Equal(t, manufactureres[1], prodInfoTwo.Manufacturer)
		assert.Equal(t, products[1], prodInfoTwo.Name)

		mockDeviceStore.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})

	t.Run("handles responses with unsupported attributes", func(t *testing.T) {
		mockZclGlobalCommunicator := mockZclGlobalCommunicator{}
		mockDeviceStore := mockDeviceStore{}

		zhpi := ZigbeeHasProductInformation{
			Gateway:               &mockGateway{},
			deviceStore:           &mockDeviceStore,
			addInternalCallback:   nil,
			zclGlobalCommunicator: &mockZclGlobalCommunicator,
		}

		node, devices := generateTestNodeAndDevices(2)
		for i := 0; i < len(devices); i++ {
			devices[i].device.Gateway = zhpi.Gateway
		}

		for _, endpoint := range node.endpoints {
			endpointDescription := node.endpointDescriptions[endpoint]
			endpointDescription.InClusterList = []zigbee.ClusterID{zcl.BasicId}
			node.endpointDescriptions[endpoint] = endpointDescription
		}

		mockDeviceStore.On("getDevice", devices[0].device.Identifier).Return(devices[0], true)
		mockDeviceStore.On("getDevice", devices[1].device.Identifier).Return(devices[1], true)

		manufactureres := []string{"manu1", "manu2"}
		products := []string{"product1", "product2"}

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(0), uint8(1), []zcl.AttributeID{0x0004, 0x0005}).
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
					Identifier:    0x0005,
					Status:        1,
					DataTypeValue: nil,
				},
			}, nil)

		mockZclGlobalCommunicator.On("ReadAttributes", mock.Anything, node.ieeeAddress, node.supportsAPSAck, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), uint8(2), []zcl.AttributeID{0x0004, 0x0005}).
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

		assert.Equal(t, []da.Capability{capabilities.HasProductInformationFlag}, devices[0].device.Capabilities)
		assert.Equal(t, []da.Capability{capabilities.HasProductInformationFlag}, devices[1].device.Capabilities)

		prodInfoOne, err := zhpi.ProductInformation(ctx, devices[0].device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Manufacturer, prodInfoOne.Present)
		assert.Equal(t, manufactureres[0], prodInfoOne.Manufacturer)

		prodInfoTwo, err := zhpi.ProductInformation(ctx, devices[1].device)
		assert.NoError(t, err)
		assert.Equal(t, capabilities.Name, prodInfoTwo.Present)
		assert.Equal(t, products[1], prodInfoTwo.Name)

		mockDeviceStore.AssertExpectations(t)
		mockZclGlobalCommunicator.AssertExpectations(t)
	})
}
