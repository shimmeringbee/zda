package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestZigbeeGateway_enableAPSACK(t *testing.T) {
	t.Run("generic zigbee device enables supportsAPSAck", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		node := &internalNode{
			mutex: &sync.RWMutex{},
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{
				IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.IEEEAddress(1), SubIdentifier: 0}: {
					productInformation: capabilities.ProductInformation{
						Present:      capabilities.Manufacturer,
						Manufacturer: "Signify",
						Name:         "",
						Serial:       "",
					},
				},
			},
		}

		err := zgw.enableAPSACK(context.Background(), internalNodeEnumeration{node: node})

		assert.NoError(t, err)
		assert.True(t, node.supportsAPSAck)
	})

	t.Run("xiaomi zigbee device disables supportsAPSAck", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		node := &internalNode{
			mutex: &sync.RWMutex{},
			nodeDesc: zigbee.NodeDescription{
				LogicalType:      0,
				ManufacturerCode: zigbee.ManufacturerCode(0x115f),
			},
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{
				IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.IEEEAddress(1), SubIdentifier: 0}: {
					productInformation: capabilities.ProductInformation{
						Present:      capabilities.Manufacturer,
						Manufacturer: "Xiaomi",
						Name:         "",
						Serial:       "",
					},
				},
			},
		}

		err := zgw.enableAPSACK(context.Background(), internalNodeEnumeration{node: node})

		assert.NoError(t, err)
		assert.False(t, node.supportsAPSAck)
	})

	t.Run("legrand zigbee device disables supportsAPSAck", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		node := &internalNode{
			mutex: &sync.RWMutex{},
			nodeDesc: zigbee.NodeDescription{
				LogicalType:      0,
				ManufacturerCode: zigbee.ManufacturerCode(0x1021),
			},
			devices: map[IEEEAddressWithSubIdentifier]*internalDevice{
				IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.IEEEAddress(1), SubIdentifier: 0}: {
					productInformation: capabilities.ProductInformation{
						Present:      capabilities.Manufacturer,
						Manufacturer: "legrand",
						Name:         "",
						Serial:       "",
					},
				},
			},
		}

		err := zgw.enableAPSACK(context.Background(), internalNodeEnumeration{node: node})

		assert.NoError(t, err)
		assert.False(t, node.supportsAPSAck)
	})
}
