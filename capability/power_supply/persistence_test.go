package power_supply

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
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					PowerStatus: capabilities.PowerStatus{
						Mains: []capabilities.PowerMainsStatus{
							{
								Voltage:   250,
								Frequency: 50.1,
								Available: true,
								Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
							},
						},
						Battery: []capabilities.PowerBatteryStatus{
							{
								Voltage:        3.2,
								NominalVoltage: 3.7,
								Remaining:      0.21,
								Available:      true,
								Present:        capabilities.Voltage | capabilities.NominalVoltage | capabilities.Remaining | capabilities.Available,
							},
						},
					},
				},
			},
			datalock: &sync.RWMutex{},
		}

		data, err := i.Save(d)
		assert.NoError(t, err)

		pd, ok := data.(*PersistentData)
		assert.True(t, ok)
		assert.Equal(t, i.data[d.Identifier].PowerStatus, pd.PowerStatus)

		_ = pd
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads state from persistence data", func(t *testing.T) {
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		d := zda.Device{
			Identifier:   zda.IEEEAddressWithSubIdentifier{IEEEAddress: addr, SubIdentifier: 0x00},
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
			Endpoints:    nil,
		}

		mockAM := mocks.MockAttributeMonitor{}
		defer mockAM.AssertExpectations(t)

		expectedData := Data{
			PowerStatus: capabilities.PowerStatus{
				Mains: []capabilities.PowerMainsStatus{
					{
						Voltage:   250,
						Frequency: 50.1,
						Available: true,
						Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
					},
				},
				Battery: []capabilities.PowerBatteryStatus{
					{
						Voltage:        3.2,
						NominalVoltage: 3.7,
						Remaining:      0.21,
						Available:      true,
						Present:        capabilities.Voltage | capabilities.NominalVoltage | capabilities.Remaining | capabilities.Available,
					},
				},
			},
		}

		pd := &PersistentData{
			PowerStatus: capabilities.PowerStatus{
				Mains: []capabilities.PowerMainsStatus{
					{
						Voltage:   250,
						Frequency: 50.1,
						Available: true,
						Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
					},
				},
				Battery: []capabilities.PowerBatteryStatus{
					{
						Voltage:        3.2,
						NominalVoltage: 3.7,
						Remaining:      0.21,
						Available:      true,
						Present:        capabilities.Voltage | capabilities.NominalVoltage | capabilities.Remaining | capabilities.Available,
					},
				},
			},
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
