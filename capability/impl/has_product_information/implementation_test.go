package has_product_information

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zda/capability/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capability.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.HasProductInformationFlag, impl.Capability())
	})
}

func TestImplementation_InitableCapability(t *testing.T) {
	t.Run("matches the InitableCapability interface", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capability.InitableCapability)(nil), impl)
	})
}

func TestImplementation_ProcessingCapability(t *testing.T) {
	t.Run("matches the ProcessingCapability interface", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capability.ProcessingCapability)(nil), impl)
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("subscribes to events", func(t *testing.T) {
		impl := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)
		defer mockEventSubscription.AssertExpectations(t)

		impl.Init(mockSupervisor)
	})
}

func TestImplementation_handleAddedDeviceRequest(t *testing.T) {
	t.Run("adding a device is added to the store, and a nil is returned on the channel", func(t *testing.T) {
		impl := &Implementation{}
		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}

		replyCh := make(chan error, 1)
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		impl.handleAddedDeviceRequest(addedDeviceReq{
			device: capability.Device{
				Identifier: addr,
			},
			ch: replyCh,
		})

		assert.Len(t, replyCh, 1)
		assert.Contains(t, impl.data, addr)
	})
}

func TestImplementation_handleRemovedDeviceRequest(t *testing.T) {
	t.Run("removing a device is removed from the store, and a nil is returned on the channel", func(t *testing.T) {
		impl := &Implementation{}

		replyCh := make(chan error, 1)
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{
			addr: {},
		}

		impl.handleRemovedDeviceRequest(removedDeviceReq{
			device: capability.Device{
				Identifier: addr,
			},
			ch: replyCh,
		})

		assert.Len(t, replyCh, 1)
		assert.NotContains(t, impl.data, addr)
	})
}

func TestImplementation_handleProductInformationReq(t *testing.T) {
	t.Run("returns a product information with both fields", func(t *testing.T) {
		impl := &Implementation{}

		replyCh := make(chan productInformationResp, 1)
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		expectedManufacturer := "manu"
		expectedProduct := "name"

		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{
			addr: {
				Manufacturer: &expectedManufacturer,
				Product:      &expectedProduct,
			},
		}

		impl.handleProductInformationReq(productInformationReq{
			device: capability.Device{
				Identifier: addr,
			},
			ch: replyCh,
		})

		assert.Len(t, replyCh, 1)

		if len(replyCh) > 0 {
			msg := <-replyCh

			assert.Equal(t, expectedManufacturer, msg.ProductInformation.Manufacturer)
			assert.Equal(t, expectedProduct, msg.ProductInformation.Name)
			assert.Equal(t, capabilities.Manufacturer|capabilities.Name, msg.ProductInformation.Present)
		}
	})
}

func TestImplementation_handleEnumerateDeviceReq(t *testing.T) {
	t.Run("removes capability if no endpoints have the Basic cluster", func(t *testing.T) {
		impl := &Implementation{}
		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := capability.Device{
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

		impl.supervisor = &capability.SimpleSupervisor{
			MDCImpl: &mockManageDeviceCapabilities,
		}

		impl.data[addr] = ProductData{
			Manufacturer: &existingManu,
			Product:      &existingProduct,
		}

		ch := make(chan error, 1)

		impl.handleEnumerateDeviceReq(enumerateDeviceReq{
			device: device,
			ch:     ch,
		})

		assert.Len(t, ch, 1)

		assert.Nil(t, impl.data[addr].Manufacturer)
		assert.Nil(t, impl.data[addr].Product)
	})

	t.Run("adds capability and sets product data if on first endpoint that has Basic cluster", func(t *testing.T) {
		impl := &Implementation{}
		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := capability.Device{
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

		impl.supervisor = &capability.SimpleSupervisor{
			MDCImpl: &mockManageDeviceCapabilities,
			ZCLImpl: &mockZCL,
		}

		impl.data[addr] = ProductData{}

		impl.msgCh = make(chan interface{}, 1)
		ch := make(chan error)

		impl.handleEnumerateDeviceReq(enumerateDeviceReq{
			device: device,
			ch:     ch,
		})

		msg := <-impl.msgCh
		enumCompMsg := msg.(enumerateDeviceComplete)

		assert.Equal(t, ch, enumCompMsg.ch)
		assert.Equal(t, device, enumCompMsg.device)
		assert.Equal(t, expectedManufacturer, *enumCompMsg.productData.Manufacturer)
		assert.Equal(t, expectedProduct, *enumCompMsg.productData.Product)
	})

	t.Run("adds capability and sets product data if on first endpoint that has Basic cluster", func(t *testing.T) {
		impl := &Implementation{}
		impl.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := capability.Device{
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

		defer mockManageDeviceCapabilities.AssertExpectations(t)

		expectedManufacturer := "manu"
		expectedProduct := "product"

		mockManageDeviceCapabilities.On("Add", device, capabilities.HasProductInformationFlag)

		impl.supervisor = &capability.SimpleSupervisor{
			MDCImpl: &mockManageDeviceCapabilities,
		}

		impl.data[addr] = ProductData{}

		ch := make(chan error, 1)

		impl.handleEnumerateDeviceComplete(enumerateDeviceComplete{
			device:      device,
			productData: ProductData{Manufacturer: &expectedManufacturer, Product: &expectedProduct},
			ch:          ch,
		})

		<-ch

		assert.Equal(t, expectedManufacturer, *impl.data[addr].Manufacturer)
		assert.Equal(t, expectedProduct, *impl.data[addr].Product)
	})
}
