package on_off

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_Off(t *testing.T) {
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

		err := i.Off(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("sends Off command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.OnOffFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]OnOffData{
			addr: {
				Endpoint:        endpoint,
				State:           true,
				RequiresPolling: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("ReadAttributes", mock.Anything, capDev, endpoint, zcl.OnOffId, []zcl.AttributeID{onoff.OnOff}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				onoff.OnOff: {
					Identifier: onoff.OnOff,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeBoolean,
						Value:    true,
					},
				},
			}, nil)

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.OnOffId, onoff.Off{}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.Off(ctx, device)

		assert.NoError(t, err)

		time.Sleep(2 * PollAfterSetDelay)
	})
}

func TestImplementation_On(t *testing.T) {
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

		err := i.On(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("sends On command to device", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		capDev := zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.OnOffFlag}}

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

		i.data = map[zda.IEEEAddressWithSubIdentifier]OnOffData{
			addr: {
				Endpoint:        endpoint,
				State:           true,
				RequiresPolling: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL.On("ReadAttributes", mock.Anything, capDev, endpoint, zcl.OnOffId, []zcl.AttributeID{onoff.OnOff}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				onoff.OnOff: {
					Identifier: onoff.OnOff,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeBoolean,
						Value:    true,
					},
				},
			}, nil)

		mockZCL.On("SendCommand", mock.Anything, capDev, endpoint, zcl.OnOffId, onoff.On{}).Return(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.On(ctx, device)

		assert.NoError(t, err)

		time.Sleep(2 * PollAfterSetDelay)
	})
}

func TestImplementation_State(t *testing.T) {
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

		_, err := i.State(ctx, device)

		assert.Error(t, err)
		assert.Equal(t, da.DeviceDoesNotBelongToGatewayError, err)
	})

	t.Run("returns data from the store for the device queried", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		device := da.BaseDevice{
			DeviceIdentifier: addr,
		}

		mockDeviceLookup := &mocks.MockDeviceLookup{}
		mockDeviceLookup.On("ByDA", device).Return(zda.Device{Identifier: addr, Capabilities: []da.Capability{capabilities.OnOffFlag}}, true)

		i := &Implementation{}
		i.supervisor = &zda.SimpleSupervisor{
			DLImpl: mockDeviceLookup,
		}

		i.data = map[zda.IEEEAddressWithSubIdentifier]OnOffData{
			addr: {
				State: true,
			},
		}
		i.datalock = &sync.RWMutex{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		state, err := i.State(ctx, device)

		assert.NoError(t, err)
		assert.True(t, state)
	})
}
