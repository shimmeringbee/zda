package has_product_information

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestImplementation_KeyName(t *testing.T) {
	t.Run("returns a name for the persistence data", func(t *testing.T) {
		i := Implementation{}

		assert.Equal(t, PersistenceName, i.KeyName())
	})
}

func TestImplementation_DataStruct(t *testing.T) {
	t.Run("returns an empty struct for the persistence data", func(t *testing.T) {
		i := Implementation{}

		s := i.DataStruct()

		_, ok := s.(*ProductData)

		assert.True(t, ok)
	})
}

func TestImplementation_Save(t *testing.T) {
	t.Run("exports the on off persistence data structure", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.HasProductInformationFlag},
			Endpoints:    nil,
		}

		expectedManu := "manu"
		expectedProduct := "product"

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]ProductData{
				d.Identifier: {
					Manufacturer: &expectedManu,
					Product:      &expectedProduct,
				},
			},
			datalock: &sync.RWMutex{},
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*ProductData)
		assert.True(t, ok)
		assert.Equal(t, &expectedManu, pd.Manufacturer)
		assert.Equal(t, &expectedProduct, pd.Product)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.HasProductInformationFlag},
			Endpoints:    nil,
		}

		expectedManu := "manu"
		expectedProduct := "product"

		expectedData := &ProductData{
			Manufacturer: &expectedManu,
			Product:      &expectedProduct,
		}

		i := Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]ProductData{},
			datalock: &sync.RWMutex{},
		}

		err := i.Load(d, expectedData)
		assert.NoError(t, err)

		state := i.data[d.Identifier]

		assert.Equal(t, *expectedData, state)
	})
}
