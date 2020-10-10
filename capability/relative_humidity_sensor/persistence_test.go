package relative_humidity_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_DataStruct(t *testing.T) {
	t.Run("returns an empty struct for the persistence data", func(t *testing.T) {
		i := Implementation{}

		s := i.DataStruct()

		_, ok := s.(*PersistentData)

		assert.True(t, ok)
	})
}

func TestImplementation_Save(t *testing.T) {
	t.Run("exports the on off persistence data structure", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.RelativeHumiditySensorFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					State:           1,
					RequiresPolling: true,
					Endpoint:        1,
				},
			},
			datalock: &sync.RWMutex{},
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*PersistentData)
		assert.True(t, ok)
		assert.Equal(t, float64(1), pd.State)
		assert.True(t, pd.RequiresPolling)
		assert.Equal(t, zigbee.Endpoint(1), pd.Endpoint)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.RelativeHumiditySensorFlag},
			Endpoints:    nil,
		}

		mockPoller := mocks.MockPoller{}
		defer mockPoller.AssertExpectations(t)
		cancelFn := func() {}
		mockPoller.On("Add", d, 5*time.Second, mock.Anything).Return(cancelFn)

		expectedData := Data{
			State:           1,
			RequiresPolling: true,
			Endpoint:        1,
		}

		pd := &PersistentData{
			State:           1,
			RequiresPolling: true,
			Endpoint:        1,
		}

		i := Implementation{
			data:       map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock:   &sync.RWMutex{},
			supervisor: &zda.SimpleSupervisor{PollerImpl: &mockPoller, DeviceConfigImpl: &mocks.DefaultDeviceConfig{}},
		}

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]

		// Can't equate functions, so need to clear it.
		assert.NotNil(t, state.PollerCancel)
		state.PollerCancel = nil

		assert.Equal(t, expectedData, state)
	})
}
