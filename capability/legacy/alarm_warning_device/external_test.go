package alarm_warning_device

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/ias_warning_device"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_pollWarningDevice(t *testing.T) {
	t.Run("sends Alarm when polling a device, that still needs an alarm", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:     mockDeviceLookup,
			ZCLImpl:    mockZCL,
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint:   endpoint,
				AlarmUntil: time.Now().Add(10 * time.Second),
				AlarmType:  capabilities.EnvironmentalAlarm,
				Volume:     0.75,
				Visual:     true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
			WarningMode:     ias_warning_device.Emergency,
			StrobeMode:      ias_warning_device.StrobeWithWarning,
			WarningDuration: 10,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		i.pollWarningDevice(ctx, capDev)
	})

	t.Run("sends Clear Alarm when polling a device, that still needs an alarm", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:     mockDeviceLookup,
			ZCLImpl:    mockZCL,
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint:   endpoint,
				AlarmUntil: time.Now().Add(-1 * time.Second),
				AlarmType:  capabilities.EnvironmentalAlarm,
				Volume:     0.75,
				Visual:     true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
			WarningMode:     ias_warning_device.Stop,
			StrobeMode:      ias_warning_device.NoStrobe,
			WarningDuration: 0,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		i.pollWarningDevice(ctx, capDev)
	})

	t.Run("sends Clear Alarm when polling a device, that no longer needs alarm, and clears the alarm", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:     mockDeviceLookup,
			ZCLImpl:    mockZCL,
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint:   endpoint,
				AlarmUntil: time.Now().Add(-1 * StopDuration * time.Second),
				AlarmType:  capabilities.EnvironmentalAlarm,
				Volume:     0.75,
				Visual:     true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
			WarningMode:     ias_warning_device.Stop,
			StrobeMode:      ias_warning_device.NoStrobe,
			WarningDuration: 0,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		i.pollWarningDevice(ctx, capDev)

		assert.True(t, i.data[addr].AlarmUntil.IsZero())
	})
}

func TestImplementation_Alarm(t *testing.T) {
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

		err := i.Alarm(ctx, device, capabilities.EnvironmentalAlarm, 0.75, true, 10*time.Second)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("sends Alarm command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:     mockDeviceLookup,
			ZCLImpl:    mockZCL,
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint: endpoint,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
			WarningMode:     ias_warning_device.Emergency,
			StrobeMode:      ias_warning_device.StrobeWithWarning,
			WarningDuration: 10,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.Alarm(ctx, device, capabilities.EnvironmentalAlarm, 0.75, true, 10*time.Second)

		assert.NoError(t, err)
		assert.Equal(t, capabilities.EnvironmentalAlarm, i.data[addr].AlarmType)
		assert.Equal(t, 0.75, i.data[addr].Volume)
		assert.True(t, i.data[addr].Visual)
		assert.NotNil(t, i.data[addr].AlarmUntil)
	})
}

func TestImplementation_Clear(t *testing.T) {
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

		err := i.Clear(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("sends Clear command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:  mockDeviceLookup,
			ZCLImpl: mockZCL,
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint:   endpoint,
				AlarmUntil: time.Now(),
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.StartWarning{
			WarningMode:     ias_warning_device.Stop,
			StrobeMode:      ias_warning_device.NoStrobe,
			WarningDuration: 0,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.Clear(ctx, device)

		assert.NoError(t, err)
		assert.True(t, i.data[addr].AlarmUntil.IsZero())
	})
}

func TestImplementation_Alert(t *testing.T) {
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

		err := i.Alert(ctx, device, capabilities.SecurityAlarm, capabilities.DisarmAlert, 0.5, true)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("sends Alert command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl:  mockDeviceLookup,
			ZCLImpl: mockZCL,
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint: endpoint,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.IASWarningDevicesId, &ias_warning_device.Squawk{
			SquawkMode:  ias_warning_device.SystemDisarmed,
			Strobe:      true,
			Reserved:    0,
			SquawkLevel: ias_warning_device.Medium,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.Alert(ctx, device, capabilities.SecurityAlarm, capabilities.DisarmAlert, 0.5, true)

		assert.NoError(t, err)
	})
}

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

	t.Run("returns status of warning device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.AlarmWarningDeviceFlag}}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(capDev, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		endpoint := zigbee.Endpoint(0x11)

		i.data = map[zda.IEEEAddressWithSubIdentifier]*Data{
			addr: {
				Endpoint:   endpoint,
				AlarmUntil: time.Now().Add(60 * time.Second),
				Volume:     0.8,
				Visual:     true,
				AlarmType:  capabilities.SecurityAlarm,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		expectedState := capabilities.WarningDeviceState{
			Warning:           true,
			AlarmType:         capabilities.SecurityAlarm,
			Volume:            0.8,
			Visual:            true,
			DurationRemaining: 60 * time.Second,
		}

		actualState, err := i.Status(ctx, device)

		if actualState.DurationRemaining > 59*time.Second {
			actualState.DurationRemaining = 60 * time.Second
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedState, actualState)
	})
}
