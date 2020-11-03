package power_supply

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
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

func TestImplementation_EnumerateDevice(t *testing.T) {
	t.Run("adds power supply information from Basic.PowerSource if available", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

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
					InClusterList: []zigbee.ClusterID{zcl.BasicId, zcl.PowerConfigurationId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.PowerSupplyFlag)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.BasicId, []zcl.AttributeID{basic.PowerSource}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				basic.PowerSource: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeEnum8,
						Value:    uint8(0x81),
					},
				},
			}, nil)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.PowerConfigurationId, []zcl.AttributeID{power_configuration.MainsVoltage, power_configuration.MainsFrequency, power_configuration.BatteryVoltage, power_configuration.BatteryPercentageRemaining, power_configuration.BatteryRatedVoltage}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				power_configuration.MainsVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt16,
						Value:    uint16(2482),
					},
				},
				power_configuration.MainsFrequency: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint8(100),
					},
				},
				power_configuration.BatteryVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint8(32),
					},
				},
				power_configuration.BatteryPercentageRemaining: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint8(100),
					},
				},
				power_configuration.BatteryRatedVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint8(37),
					},
				},
			}, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			ZCLImpl:          &mockZCL,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, true, i.data[addr].PowerStatus.Battery[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.NominalVoltage|capabilities.Remaining, i.data[addr].PowerStatus.Battery[0].Present)
		assert.Equal(t, 3.2, i.data[addr].PowerStatus.Battery[0].Voltage)
		assert.Equal(t, 3.7, i.data[addr].PowerStatus.Battery[0].NominalVoltage)
		assert.Equal(t, 50.0, i.data[addr].PowerStatus.Battery[0].Remaining)

		assert.Equal(t, true, i.data[addr].PowerStatus.Mains[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.Frequency, i.data[addr].PowerStatus.Mains[0].Present)
		assert.Equal(t, 248.2, i.data[addr].PowerStatus.Mains[0].Voltage)
		assert.Equal(t, 50.0, i.data[addr].PowerStatus.Mains[0].Frequency)
	})
}
