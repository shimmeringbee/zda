package power_supply

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

func TestImplementation_Status(t *testing.T) {
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

		_, err := i.Status(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns data from the store for the device queried", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.PowerSupplyFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
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
						MinimumVoltage: 3.1,
						Remaining:      0.21,
						Available:      true,
						Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.MinimumVoltage | capabilities.Remaining | capabilities.Available,
					},
				},
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		expectedState := capabilities.PowerStatus{
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
					MinimumVoltage: 3.1,
					Remaining:      0.21,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.MinimumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
		}

		actualState, err := i.Status(ctx, device)

		assert.NoError(t, err)
		assert.Equal(t, expectedState, actualState)
	})

	t.Run("returned data includes synthetic remaining if all voltages on a battery are present", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.PowerSupplyFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Battery: []*capabilities.PowerBatteryStatus{
					{
						Voltage:        7.5,
						MaximumVoltage: 10,
						MinimumVoltage: 5,
						Available:      true,
						Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.MinimumVoltage | capabilities.Available,
					},
				},
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		expectedState := capabilities.PowerStatus{
			Battery: []capabilities.PowerBatteryStatus{
				{
					Voltage:        7.5,
					MaximumVoltage: 10,
					MinimumVoltage: 5,
					Remaining:      0.5,
					Available:      true,
					Present:        capabilities.Voltage | capabilities.MaximumVoltage | capabilities.MinimumVoltage | capabilities.Remaining | capabilities.Available,
				},
			},
		}

		actualState, err := i.Status(ctx, device)

		assert.NoError(t, err)
		assert.Equal(t, expectedState, actualState)
	})
}

func TestImplementation_LastChangeTime(t *testing.T) {
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

		_, err := i.LastChangeTime(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns data from the store for the device queried", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.PowerSupplyFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		expectedTime := time.Now()

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				LastChangeTime: expectedTime,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		actualTime, err := i.LastChangeTime(ctx, device)

		assert.NoError(t, err)
		assert.Equal(t, expectedTime, actualTime)
	})
}

func TestImplementation_LastUpdateTime(t *testing.T) {
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

		_, err := i.LastUpdateTime(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns data from the store for the device queried", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.PowerSupplyFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		expectedTime := time.Now()

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				LastUpdateTime: expectedTime,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		actualTime, err := i.LastUpdateTime(ctx, device)

		assert.NoError(t, err)
		assert.Equal(t, expectedTime, actualTime)
	})
}
