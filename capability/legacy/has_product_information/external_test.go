package has_product_information

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestImplementation_CapabilityInterface(t *testing.T) {
	t.Run("matches the HasProductInformation interface", func(t *testing.T) {
		i := &Implementation{}

		assert.Implements(t, (*capabilities.HasProductInformation)(nil), i)
	})
}

func TestImplementation_ProductInformation(t *testing.T) {
	t.Run("querying for data returns error if device is not in store", func(t *testing.T) {
		device := da.BaseDevice{
			DeviceIdentifier: zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00},
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{}, false)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := i.ProductInformation(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns a product information with both fields", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{
			Identifier:   device.DeviceIdentifier.(zda.IEEEAddressWithSubIdentifier),
			Capabilities: []da.Capability{capabilities.HasProductInformationFlag},
		}, true)

		i := &Implementation{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		expectedManufacturer := "manu"
		expectedProduct := "name"

		i.data = map[zda.IEEEAddressWithSubIdentifier]ProductData{
			addr: {
				Manufacturer: &expectedManufacturer,
				Product:      &expectedProduct,
			},
		}

		pi, err := i.ProductInformation(context.Background(), device)
		assert.NoError(t, err)

		assert.Equal(t, expectedManufacturer, pi.Manufacturer)
		assert.Equal(t, expectedProduct, pi.Name)
		assert.Equal(t, capabilities.Manufacturer|capabilities.Name, pi.Present)
	})
}
