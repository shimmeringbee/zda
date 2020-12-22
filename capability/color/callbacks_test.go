package color

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_addedDeviceCallback(t *testing.T) {
	t.Run("adding a device is added to the store, and a nil is returned on the channel", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.AddedDevice(ctx, device)

		assert.NoError(t, err)
		assert.Contains(t, i.data, id)
	})
}

func TestImplementation_removedDeviceCallback(t *testing.T) {
	t.Run("removing a device is removed from the store, and a nil is returned on the channel", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		i.data[id] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.RemovedDevice(ctx, device)

		assert.NoError(t, err)
		assert.NotContains(t, i.data, id)
	})
}

func TestImplementation_enumerateDeviceCallback(t *testing.T) {
	t.Run("removes capability if no endpoints have the Color cluster", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

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

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		endpoint := zigbee.Endpoint(0)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{},
				},
			},
		}

		mockColorMode.On("Detach", mock.Anything, device)
		mockCurrentX.On("Detach", mock.Anything, device)
		mockCurrentY.On("Detach", mock.Anything, device)
		mockCurrentHue.On("Detach", mock.Anything, device)
		mockCurrentSat.On("Detach", mock.Anything, device)
		mockCurrentTemp.On("Detach", mock.Anything, device)

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Remove", device, capabilities.ColorFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{
			State: State{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, State{}, i.data[addr].State)
	})

	t.Run("adds capability and sets product data if on first endpoint that has Color cluster, puts requires polling in data", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

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

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		endpoint := zigbee.Endpoint(0x01)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockColorMode.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)
		mockCurrentX.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)
		mockCurrentY.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)
		mockCurrentHue.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)
		mockCurrentSat.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)
		mockCurrentTemp.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.ColorFlag)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.ColorControlId, []zcl.AttributeID{0x400a}).Return(map[zcl.AttributeID]global.ReadAttributeResponseRecord{
			0x400a: {
				Identifier: 0,
				Status:     0,
				DataTypeValue: &zcl.AttributeDataTypeValue{
					DataType: zcl.TypeEnum16,
					Value:    uint64(0b00011001),
				},
			}}, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			LoggerImpl:       logwrap.New(discard.Discard()),
			ZCLImpl:          &mockZCL,
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.True(t, i.data[addr].SupportsTemperature)
		assert.True(t, i.data[addr].SupportsHueSat)
		assert.True(t, i.data[addr].SupportsXY)
		//assert.True(t, i.data[addr].RequiresPolling)
	})
}
