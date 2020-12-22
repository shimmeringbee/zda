package color

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
			Capabilities: []da.Capability{capabilities.ColorFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					State: State{
						CurrentMode:        1,
						CurrentX:           1.0,
						CurrentY:           2.0,
						CurrentHue:         180.0,
						CurrentSat:         0.7,
						CurrentTemperature: 1600,
					},
					RequiresPolling:     true,
					Endpoint:            1,
					SupportsXY:          true,
					SupportsHueSat:      true,
					SupportsTemperature: true,
				},
			},
			datalock: &sync.RWMutex{},
		}

		expectedState := PersistentState{
			CurrentMode:        1,
			CurrentX:           1.0,
			CurrentY:           2.0,
			CurrentHue:         180.0,
			CurrentSat:         0.7,
			CurrentTemperature: 1600,
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*PersistentData)
		assert.True(t, ok)
		assert.Equal(t, expectedState, pd.State)
		assert.True(t, pd.RequiresPolling)
		assert.True(t, pd.SupportsXY)
		assert.True(t, pd.SupportsHueSat)
		assert.True(t, pd.SupportsTemperature)
		assert.Equal(t, zigbee.Endpoint(1), pd.Endpoint)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.ColorFlag},
			Endpoints:    nil,
		}

		expectedData := Data{
			State: State{
				CurrentMode:        1,
				CurrentX:           1.0,
				CurrentY:           2.0,
				CurrentHue:         180.0,
				CurrentSat:         0.7,
				CurrentTemperature: 1600,
			},
			RequiresPolling:     true,
			Endpoint:            1,
			SupportsXY:          true,
			SupportsHueSat:      true,
			SupportsTemperature: true,
		}

		pd := &PersistentData{
			State: PersistentState{
				CurrentMode:        1,
				CurrentX:           1.0,
				CurrentY:           2.0,
				CurrentHue:         180.0,
				CurrentSat:         0.7,
				CurrentTemperature: 1600,
			},
			RequiresPolling:     true,
			Endpoint:            1,
			SupportsXY:          true,
			SupportsHueSat:      true,
			SupportsTemperature: true,
		}

		i := Implementation{
			data:       map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock:   &sync.RWMutex{},
			supervisor: &zda.SimpleSupervisor{DeviceConfigImpl: &mocks.DefaultDeviceConfig{}},
		}

		mockColorMode := mocks.MockAttributeMonitor{}
		i.attMonColorMode = &mockColorMode
		defer mockColorMode.AssertExpectations(t)

		mockCurrentX := mocks.MockAttributeMonitor{}
		i.attMonCurrentX = &mockCurrentX
		defer mockCurrentX.AssertExpectations(t)

		mockCurrentY := mocks.MockAttributeMonitor{}
		i.attMonCurrentY = &mockCurrentY
		defer mockCurrentY.AssertExpectations(t)

		mockCurrentHue := mocks.MockAttributeMonitor{}
		i.attMonCurrentHue = &mockCurrentHue
		defer mockCurrentHue.AssertExpectations(t)

		mockCurrentSat := mocks.MockAttributeMonitor{}
		i.attMonCurrentSat = &mockCurrentSat
		defer mockCurrentSat.AssertExpectations(t)

		mockCurrentTemp := mocks.MockAttributeMonitor{}
		i.attMonCurrentTemp = &mockCurrentTemp
		defer mockCurrentTemp.AssertExpectations(t)

		mockColorMode.On("Reattach", mock.Anything, d, pd.Endpoint, true)
		mockCurrentX.On("Reattach", mock.Anything, d, pd.Endpoint, true)
		mockCurrentY.On("Reattach", mock.Anything, d, pd.Endpoint, true)
		mockCurrentHue.On("Reattach", mock.Anything, d, pd.Endpoint, true)
		mockCurrentSat.On("Reattach", mock.Anything, d, pd.Endpoint, true)
		mockCurrentTemp.On("Reattach", mock.Anything, d, pd.Endpoint, true)

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]
		assert.Equal(t, expectedData, state)
	})
}
