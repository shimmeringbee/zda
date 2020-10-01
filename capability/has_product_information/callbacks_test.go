package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_addedDeviceCallback(t *testing.T) {
	t.Run("adding a device is added to the store, and a nil is returned on the channel", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}
		i.datalock = &sync.RWMutex{}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.AddedDevice(ctx, device)

		assert.NoError(t, err)
		assert.Contains(t, i.data, id)
	})
}

func TestImplementation_removedDeviceCallback(t *testing.T) {
	t.Run("removing a device is removed from the store, and a nil is returned on the channel", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}
		i.datalock = &sync.RWMutex{}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		i.data[id] = ProductData{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.RemovedDevice(ctx, device)

		assert.NoError(t, err)
		assert.NotContains(t, i.data, id)
	})
}

func TestImplementation_enumerateDeviceCallback(t *testing.T) {
	t.Run("removes capability if no endpoints have the Basic cluster", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}
		i.datalock = &sync.RWMutex{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				0x00: {
					Endpoint:      0x00,
					InClusterList: []zigbee.ClusterID{},
				},
			},
		}

		existingManu := "existingManu"
		existingProduct := "existingProduct"

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Remove", device, capabilities.HasProductInformationFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl: &mockManageDeviceCapabilities,
		}

		i.data[addr] = ProductData{
			Manufacturer: &existingManu,
			Product:      &existingProduct,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Nil(t, i.data[addr].Manufacturer)
		assert.Nil(t, i.data[addr].Product)
	})

	t.Run("adds capability and sets product data if on first endpoint that has Basic cluster", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}
		i.datalock = &sync.RWMutex{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				0x00: {
					Endpoint:      0x00,
					InClusterList: []zigbee.ClusterID{zcl.BasicId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		mockZCL := mocks.MockZCL{}

		defer mockManageDeviceCapabilities.AssertExpectations(t)
		defer mockZCL.AssertExpectations(t)

		expectedManufacturer := "manu"
		expectedProduct := "product"

		mockZCL.On("ReadAttributes", mock.Anything, device, zigbee.Endpoint(0x00), zcl.BasicId, []zcl.AttributeID{0x0004, 0x0005}).
			Return(map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				0x0004: {
					Status: 0x00,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    &expectedManufacturer,
					},
				},
				0x0005: {
					Status: 0x00,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeStringCharacter8,
						Value:    &expectedProduct,
					},
				},
			}, nil)

		mockManageDeviceCapabilities.On("Add", device, capabilities.HasProductInformationFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl: &mockManageDeviceCapabilities,
			ZCLImpl: &mockZCL,
		}

		i.data[addr] = ProductData{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, expectedManufacturer, *i.data[addr].Manufacturer)
		assert.Equal(t, expectedProduct, *i.data[addr].Product)
	})
}
