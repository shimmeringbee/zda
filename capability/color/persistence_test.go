package color

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
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
			Capabilities: []da.Capability{capabilities.ColorFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					State: State{
						CurrentMode:        capabilities.ColorMode,
						CurrentColor:       color.SRGBColor{R: 255, G: 192, B: 64},
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
			CurrentMode: capabilities.ColorMode,
			CurrentColor: PersistentColor{
				ColorSpace: color.SRGB,
				R:          255,
				G:          192,
				B:          64,
			},
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
				CurrentMode:        capabilities.ColorMode,
				CurrentColor:       color.SRGBColor{R: 255, G: 192, B: 64},
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
				CurrentMode: capabilities.ColorMode,
				CurrentColor: PersistentColor{
					ColorSpace: color.SRGB,
					R:          255,
					G:          192,
					B:          64,
				},
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

		//mockAM.On("Reattach", mock.Anything, d, pd.Endpoint, true)

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]
		assert.Equal(t, expectedData, state)
	})
}
