package power_supply

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
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
			Endpoints:    nil,
		}

		i := Implementation{
			data: map[zda.IEEEAddressWithSubIdentifier]Data{
				d.Identifier: {
					Mains: []*capabilities.PowerMainsStatus{
						{
							Voltage:   250,
							Frequency: 50.1,
							Available: true,
							Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
						},
					},
					Battery: []*capabilities.PowerBatteryStatus{
						{
							Voltage:        3.2,
							MaximumVoltage: 3.7,
							Remaining:      0.21,
							Available:      true,
							Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.Remaining | capabilities.Available,
						},
					},
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
		assert.Equal(t, i.data[d.Identifier].Mains[0], &pd.Mains[0])
		assert.Equal(t, i.data[d.Identifier].Battery[0], &pd.Battery[0])
		assert.True(t, pd.RequiresPolling)
		assert.Equal(t, zigbee.Endpoint(1), pd.Endpoint)

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

		expectedData := Data{
			Mains: []*capabilities.PowerMainsStatus{
				{
					Voltage:   250,
					Frequency: 50.1,
					Available: true,
					Present:   capabilities.Voltage | capabilities.Frequency | capabilities.Available,
				},
			},
			Battery: []*capabilities.PowerBatteryStatus{
				{
					Voltage:        3.2,
					MaximumVoltage: 3.7,
					Remaining:      0.21,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
			RequiresPolling:    true,
			Endpoint:           1,
			PowerConfiguration: true,
		}

		pd := &PersistentData{
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
					MaximumVoltage: 3.7,
					Remaining:      0.21,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
			RequiresPolling:    true,
			Endpoint:           1,
			PowerConfiguration: true,
		}

		mockAMMainsVoltage := mocks.MockAttributeMonitor{}
		defer mockAMMainsVoltage.AssertExpectations(t)
		mockAMMainsFrequency := mocks.MockAttributeMonitor{}
		defer mockAMMainsFrequency.AssertExpectations(t)
		mockAMBatteryVoltage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryVoltage.AssertExpectations(t)
		mockAMBatteryRemainingPercentage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryRemainingPercentage.AssertExpectations(t)

		mockAMMainsVoltage.On("Reattach", mock.Anything, d, pd.Endpoint, pd.RequiresPolling)
		mockAMMainsFrequency.On("Reattach", mock.Anything, d, pd.Endpoint, pd.RequiresPolling)
		mockAMBatteryVoltage.On("Reattach", mock.Anything, d, pd.Endpoint, pd.RequiresPolling)
		mockAMBatteryRemainingPercentage.On("Reattach", mock.Anything, d, pd.Endpoint, pd.RequiresPolling)

		i := Implementation{
			data:                             map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock:                         &sync.RWMutex{},
			supervisor:                       &zda.SimpleSupervisor{DeviceConfigImpl: &mocks.DefaultDeviceConfig{}},
			attMonMainsVoltage:               &mockAMMainsVoltage,
			attMonMainsFrequency:             &mockAMMainsFrequency,
			attMonBatteryVoltage:             &mockAMBatteryVoltage,
			attMonBatteryPercentageRemaining: &mockAMBatteryRemainingPercentage,
		}

		err := i.Load(d, pd)
		assert.NoError(t, err)

		state := i.data[d.Identifier]
		assert.Equal(t, expectedData, state)
	})
}
