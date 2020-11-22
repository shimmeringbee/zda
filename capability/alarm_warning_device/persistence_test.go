package alarm_warning_device

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
			Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]*Data{
				d.Identifier: {
					Endpoint: 1,
				},
			},
			datalock: &sync.RWMutex{},
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*PersistentData)
		assert.True(t, ok)
		assert.Equal(t, zigbee.Endpoint(1), pd.Endpoint)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag},
			Endpoints:    nil,
		}

		pd := &PersistentData{
			Endpoint: 1,
		}

		mockPoller := mocks.MockPoller{}
		defer mockPoller.AssertExpectations(t)

		retFunc := func() {}
		mockPoller.On("Add", d, AnnouncementPeriod, mock.Anything).Return(retFunc)

		i := Implementation{
			data:       map[zda.IEEEAddressWithSubIdentifier]*Data{},
			datalock:   &sync.RWMutex{},
			supervisor: zda.SimpleSupervisor{PollerImpl: &mockPoller},
		}

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]
		assert.Equal(t, zigbee.Endpoint(1), state.Endpoint)
		assert.NotNil(t, state.PollerCancel)
	})
}
