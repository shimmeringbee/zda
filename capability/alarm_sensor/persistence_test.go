package alarm_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
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
			Capabilities: []da.Capability{capabilities.AlarmSensorFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					Alarms: map[capabilities.SensorType]bool{
						capabilities.Radiation: true,
					},
					Endpoint: zigbee.Endpoint(0x02),
					ZoneType: 0x0003,
				},
			},
			datalock: &sync.RWMutex{},
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*PersistentData)
		assert.True(t, ok)
		assert.True(t, pd.Alarms["Radiation"])
		assert.Equal(t, zigbee.Endpoint(0x02), pd.Endpoint)
		assert.Equal(t, uint16(3), pd.ZoneType)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.AlarmSensorFlag},
			Endpoints:    nil,
		}

		mockAM := mocks.MockAttributeMonitor{}
		defer mockAM.AssertExpectations(t)

		expectedData := Data{
			Alarms: map[capabilities.SensorType]bool{
				capabilities.Radiation: true,
			},
			Endpoint: 0x02,
			ZoneType: 0x0003,
		}

		pd := &PersistentData{
			Alarms: map[string]bool{
				"Radiation": true,
			},
			Endpoint: 0x02,
			ZoneType: 0x0003,
		}

		i := Implementation{
			data:       map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock:   &sync.RWMutex{},
			supervisor: &zda.SimpleSupervisor{DeviceConfigImpl: &mocks.DefaultDeviceConfig{}},
		}

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]
		assert.Equal(t, expectedData, state)
	})
}
