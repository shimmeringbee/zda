package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zda/capability/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestImplementation_CapabilityInterface(t *testing.T) {
	t.Run("matches the HasProductInformation interface", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capabilities.HasProductInformation)(nil), impl)
	})
}

func TestImplementation_ProductInformation(t *testing.T) {
	t.Run("querying for data returns error if device is not in store", func(t *testing.T) {
		device := da.BaseDevice{
			DeviceIdentifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00},
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capability.Device{}, false)

		i := &Implementation{}
		i.supervisor = &capability.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := i.ProductInformation(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		device := da.BaseDevice{
			DeviceIdentifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00},
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capability.Device{Capabilities: []da.Capability{capabilities.HasProductInformationFlag}}, true)

		i := &Implementation{}
		i.supervisor = &capability.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := i.ProductInformation(ctx, device)
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("querying for data returns value if channel receives reply", func(t *testing.T) {
		device := da.BaseDevice{
			DeviceIdentifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00},
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capability.Device{Capabilities: []da.Capability{capabilities.HasProductInformationFlag}}, true)

		i := &Implementation{}
		i.supervisor = &capability.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		expectedPi := capabilities.ProductInformation{
			Present:      capabilities.Name | capabilities.Manufacturer,
			Manufacturer: "manu",
			Name:         "product",
		}

		go func() {
			msg := (<- i.msgCh).(productInformationReq)
			msg.ch <- productInformationResp{
				ProductInformation: expectedPi,
			}
		}()

		actualPi, err := i.ProductInformation(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, expectedPi, actualPi)
	})
}
