package color

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_ChangeColor(t *testing.T) {
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

		err := i.ChangeColor(ctx, device, color.XYColor{
			X:  0.4,
			Y:  0.4,
			Y2: 100,
		}, 100*time.Millisecond)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns error if device has Color capability, but does not support Color", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Endpoint:        endpoint,
				RequiresPolling: true,
				SupportsXY:      false,
				SupportsHueSat:  false,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.ChangeColor(ctx, device, color.XYColor{
			X:  0.4,
			Y:  0.4,
			Y2: 100,
		}, 100*time.Millisecond)

		assert.Error(t, err)
	})

	t.Run("sends MoveToHueAndSaturation command to device if it has SupportHueSat and a HueSat color is provided", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Endpoint:        endpoint,
				SupportsHueSat:  true,
				RequiresPolling: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.ColorControlId, &color_control.MoveToHueAndSaturation{
			Hue:            127,
			Saturation:     127,
			TransitionTime: 3,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.ChangeColor(ctx, device, color.HSVColor{
			Hue:   180,
			Sat:   0.5,
			Value: 1.0,
		}, 300*time.Millisecond)

		assert.NoError(t, err)

		time.Sleep(2 * PollAfterSetDelay)
	})

	t.Run("sends MoveToColor command to device if it has SupportXY and a XY color is provided", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Endpoint:        endpoint,
				SupportsXY:      true,
				RequiresPolling: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.ColorControlId, &color_control.MoveToColor{
			ColorX:         32768,
			ColorY:         32768,
			TransitionTime: 3,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.ChangeColor(ctx, device, color.XYColor{
			X:  0.5,
			Y:  0.5,
			Y2: 100.0,
		}, 300*time.Millisecond)

		assert.NoError(t, err)

		time.Sleep(2 * PollAfterSetDelay)
	})
}

func TestImplementation_ChangeTemperature(t *testing.T) {
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

		err := i.ChangeTemperature(ctx, device, 6500, 100*time.Millisecond)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns error if device has Color capability, but does not support Temperature", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Endpoint:            endpoint,
				RequiresPolling:     true,
				SupportsTemperature: false,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.ChangeTemperature(ctx, device, 6500, 100*time.Millisecond)

		assert.Error(t, err)
	})

	t.Run("sends MoveToColorTemperature command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				Endpoint:            endpoint,
				SupportsTemperature: true,
				RequiresPolling:     true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.ColorControlId, &color_control.MoveToColorTemperature{
			ColorTemperatureMireds: 154,
			TransitionTime:         3,
		}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.ChangeTemperature(ctx, device, 6500, 300*time.Millisecond)

		assert.NoError(t, err)

		time.Sleep(2 * PollAfterSetDelay)
	})
}

func TestImplementation_SupportsColor(t *testing.T) {
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

		_, err := i.SupportsColor(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns true if the device supports HueSat", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				SupportsHueSat: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		state, err := i.SupportsColor(ctx, device)

		assert.NoError(t, err)
		assert.True(t, state)
	})

	t.Run("returns true if the device supports XY", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				SupportsXY: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		state, err := i.SupportsColor(ctx, device)

		assert.NoError(t, err)
		assert.True(t, state)
	})
}

func TestImplementation_SupportsTemperature(t *testing.T) {
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

		_, err := i.SupportsTemperature(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns true if the device supports HueSat", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				SupportsTemperature: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		state, err := i.SupportsTemperature(ctx, device)

		assert.NoError(t, err)
		assert.True(t, state)
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

	t.Run("returns true if the device supports HueSat", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.ColorFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        2,
					CurrentTemperature: 2500,
				},
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		expectedState := capabilities.ColorStatus{
			Mode: capabilities.TemperatureMode,
			Temperature: capabilities.TemperatureSettings{
				Current: 2500.0,
			},
		}
		actualState, err := i.Status(ctx, device)

		assert.NoError(t, err)
		assert.Equal(t, expectedState, actualState)
	})
}
