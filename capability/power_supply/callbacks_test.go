package power_supply

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
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
		mockAMMainsVoltage := mocks.MockAttributeMonitor{}
		defer mockAMMainsVoltage.AssertExpectations(t)
		mockAMMainsFrequency := mocks.MockAttributeMonitor{}
		defer mockAMMainsFrequency.AssertExpectations(t)
		mockAMBatteryVoltage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryVoltage.AssertExpectations(t)
		mockAMBatteryRemainingPercentage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryRemainingPercentage.AssertExpectations(t)
		mockAMVendorXiaomiApproachOne := mocks.MockAttributeMonitor{}
		defer mockAMVendorXiaomiApproachOne.AssertExpectations(t)

		i := &Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock: &sync.RWMutex{},

			attMonMainsVoltage:               &mockAMMainsVoltage,
			attMonMainsFrequency:             &mockAMMainsFrequency,
			attMonBatteryVoltage:             &mockAMBatteryVoltage,
			attMonBatteryPercentageRemaining: &mockAMBatteryRemainingPercentage,
			attMonVendorXiaomiApproachOne:    &mockAMVendorXiaomiApproachOne,
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		i.data[id] = Data{}

		mockAMMainsVoltage.On("Detach", mock.Anything, device)
		mockAMMainsFrequency.On("Detach", mock.Anything, device)
		mockAMBatteryVoltage.On("Detach", mock.Anything, device)
		mockAMBatteryRemainingPercentage.On("Detach", mock.Anything, device)
		mockAMVendorXiaomiApproachOne.On("Detach", mock.Anything, device)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.RemovedDevice(ctx, device)

		assert.NoError(t, err)
		assert.NotContains(t, i.data, id)
	})
}

func TestImplementation_EnumerateDevice(t *testing.T) {
	t.Run("adds power supply information from Basic.PowerSource with Enum8 value and PowerConfiguration if available", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockAMMainsVoltage := mocks.MockAttributeMonitor{}
		defer mockAMMainsVoltage.AssertExpectations(t)
		mockAMMainsFrequency := mocks.MockAttributeMonitor{}
		defer mockAMMainsFrequency.AssertExpectations(t)
		mockAMBatteryVoltage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryVoltage.AssertExpectations(t)
		mockAMBatteryRemainingPercentage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryRemainingPercentage.AssertExpectations(t)

		i := &Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock: &sync.RWMutex{},

			attMonMainsVoltage:               &mockAMMainsVoltage,
			attMonMainsFrequency:             &mockAMMainsFrequency,
			attMonBatteryVoltage:             &mockAMBatteryVoltage,
			attMonBatteryPercentageRemaining: &mockAMBatteryRemainingPercentage,
		}

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
						Value:    uint64(2482),
					},
				},
				power_configuration.MainsFrequency: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(100),
					},
				},
				power_configuration.BatteryVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(32),
					},
				},
				power_configuration.BatteryPercentageRemaining: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(100),
					},
				},
				power_configuration.BatteryRatedVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(37),
					},
				},
			}, nil)

		mockAMMainsVoltage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMMainsFrequency.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMBatteryVoltage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMBatteryRemainingPercentage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			ZCLImpl:          &mockZCL,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, true, i.data[addr].Battery[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.MaximumVoltage|capabilities.Remaining, i.data[addr].Battery[0].Present)
		assert.Equal(t, 3.2, i.data[addr].Battery[0].Voltage)
		assert.Equal(t, 3.7, i.data[addr].Battery[0].MaximumVoltage)
		assert.Equal(t, 0.5, i.data[addr].Battery[0].Remaining)

		assert.Equal(t, true, i.data[addr].Mains[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.Frequency, i.data[addr].Mains[0].Present)
		assert.Equal(t, 248.2, i.data[addr].Mains[0].Voltage)
		assert.Equal(t, 50.0, i.data[addr].Mains[0].Frequency)

		assert.True(t, i.data[addr].PowerConfiguration)
	})

	t.Run("adds power supply information from Basic.PowerSource with Uint8 value and PowerConfiguration if available", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockAMMainsVoltage := mocks.MockAttributeMonitor{}
		defer mockAMMainsVoltage.AssertExpectations(t)
		mockAMMainsFrequency := mocks.MockAttributeMonitor{}
		defer mockAMMainsFrequency.AssertExpectations(t)
		mockAMBatteryVoltage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryVoltage.AssertExpectations(t)
		mockAMBatteryRemainingPercentage := mocks.MockAttributeMonitor{}
		defer mockAMBatteryRemainingPercentage.AssertExpectations(t)

		i := &Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock: &sync.RWMutex{},

			attMonMainsVoltage:               &mockAMMainsVoltage,
			attMonMainsFrequency:             &mockAMMainsFrequency,
			attMonBatteryVoltage:             &mockAMBatteryVoltage,
			attMonBatteryPercentageRemaining: &mockAMBatteryRemainingPercentage,
		}

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
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(0x81),
					},
				},
			}, nil)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.PowerConfigurationId, []zcl.AttributeID{power_configuration.MainsVoltage, power_configuration.MainsFrequency, power_configuration.BatteryVoltage, power_configuration.BatteryPercentageRemaining, power_configuration.BatteryRatedVoltage}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				power_configuration.MainsVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt16,
						Value:    uint64(2482),
					},
				},
				power_configuration.MainsFrequency: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(100),
					},
				},
				power_configuration.BatteryVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(32),
					},
				},
				power_configuration.BatteryPercentageRemaining: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(100),
					},
				},
				power_configuration.BatteryRatedVoltage: {
					Status: 0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt8,
						Value:    uint64(37),
					},
				},
			}, nil)

		mockAMMainsVoltage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMMainsFrequency.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMBatteryVoltage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)
		mockAMBatteryRemainingPercentage.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			ZCLImpl:          &mockZCL,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, true, i.data[addr].Battery[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.MaximumVoltage|capabilities.Remaining, i.data[addr].Battery[0].Present)
		assert.Equal(t, 3.2, i.data[addr].Battery[0].Voltage)
		assert.Equal(t, 3.7, i.data[addr].Battery[0].MaximumVoltage)
		assert.Equal(t, 0.5, i.data[addr].Battery[0].Remaining)

		assert.Equal(t, true, i.data[addr].Mains[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage|capabilities.Frequency, i.data[addr].Mains[0].Present)
		assert.Equal(t, 248.2, i.data[addr].Mains[0].Voltage)
		assert.Equal(t, 50.0, i.data[addr].Mains[0].Frequency)

		assert.True(t, i.data[addr].PowerConfiguration)
	})

	t.Run("adds power supply information from Xaiomi vendor specific attribute if available", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockAMXiaomi := mocks.MockAttributeMonitor{}
		defer mockAMXiaomi.AssertExpectations(t)

		i := &Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock: &sync.RWMutex{},

			attMonVendorXiaomiApproachOne: &mockAMXiaomi,
		}

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
					InClusterList: []zigbee.ClusterID{zcl.BasicId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.PowerSupplyFlag)

		mockConfig := mocks.MockConfig{}
		defer mockConfig.AssertExpectations(t)
		mockConfig.On("Int", "PowerConfigurationEndpoint", mock.Anything).Return(1)
		mockConfig.On("Bool", "HasBasicPowerSource", mock.Anything).Return(false)
		mockConfig.On("Bool", "HasPowerConfiguration", mock.Anything).Return(false)
		mockConfig.On("Bool", "HasVendorXiaomiApproachOne", mock.Anything).Return(true)
		mockConfig.On("Float", "MinimumVoltage", mock.Anything).Return(0.0)
		mockConfig.On("Float", "MaximumVoltage", mock.Anything).Return(0.0)

		mockDeviceConfig := mocks.MockDeviceConfig{}
		defer mockDeviceConfig.AssertExpectations(t)
		mockDeviceConfig.On("Get", device, "PowerSupply").Return(&mockConfig)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mockDeviceConfig,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		mockAMXiaomi.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, true, i.data[addr].Battery[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.Voltage, i.data[addr].Battery[0].Present)
		assert.True(t, i.data[addr].VendorXiaomiApproachOne)
	})

	t.Run("adds power supply information from Basic.PowerSource and populates minimum and maximum voltage", func(t *testing.T) {
		mockZCL := mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		i := &Implementation{
			data:     map[zda.IEEEAddressWithSubIdentifier]Data{},
			datalock: &sync.RWMutex{},
		}

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
					InClusterList: []zigbee.ClusterID{zcl.BasicId},
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

		mockConfig := mocks.MockConfig{}
		defer mockConfig.AssertExpectations(t)
		mockConfig.On("Int", "PowerConfigurationEndpoint", mock.Anything).Return(1)
		mockConfig.On("Int", "BasicEndpoint", mock.Anything).Return(1)

		mockConfig.On("Bool", "HasBasicPowerSource", mock.Anything).Return(true)
		mockConfig.On("Bool", "HasPowerConfiguration", mock.Anything).Return(false)
		mockConfig.On("Bool", "HasVendorXiaomiApproachOne", mock.Anything).Return(false)

		mockConfig.On("Float", "MinimumVoltage", mock.Anything).Return(3.4)
		mockConfig.On("Float", "MaximumVoltage", mock.Anything).Return(4.2)

		mockDeviceConfig := mocks.MockDeviceConfig{}
		defer mockDeviceConfig.AssertExpectations(t)
		mockDeviceConfig.On("Get", device, "PowerSupply").Return(&mockConfig)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mockDeviceConfig,
			ZCLImpl:          &mockZCL,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, true, i.data[addr].Battery[0].Available)
		assert.Equal(t, capabilities.Available|capabilities.MaximumVoltage|capabilities.MinimumVoltage, i.data[addr].Battery[0].Present)
		assert.Equal(t, 3.4, i.data[addr].Battery[0].MinimumVoltage)
		assert.Equal(t, 4.2, i.data[addr].Battery[0].MaximumVoltage)
	})
}

func TestImplementation_attributeUpdateMainsVoltage(t *testing.T) {
	t.Run("updating mains voltage via attribute updates data store", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   id,
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		i.data[device.Identifier] = Data{
			Mains: []*capabilities.PowerMainsStatus{
				{
					Present: capabilities.Voltage,
				},
			},
		}

		currentTime := time.Now()

		i.attributeUpdateMainsVoltage(device, 0, zcl.AttributeDataTypeValue{
			DataType: 0,
			Value:    uint64(2455),
		})

		assert.Equal(t, 245.5, i.data[device.Identifier].Mains[0].Voltage)
		assert.True(t, i.data[device.Identifier].LastChangeTime.Equal(currentTime) || i.data[device.Identifier].LastChangeTime.After(currentTime))
		assert.True(t, i.data[device.Identifier].LastUpdateTime.Equal(currentTime) || i.data[device.Identifier].LastUpdateTime.After(currentTime))
	})
}

func TestImplementation_attributeUpdateMainsFrequency(t *testing.T) {
	t.Run("updating mains frequency via attribute updates data store", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   id,
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		i.data[device.Identifier] = Data{
			Mains: []*capabilities.PowerMainsStatus{
				{
					Present: capabilities.Frequency,
				},
			},
		}

		currentTime := time.Now()

		i.attributeUpdateMainsFrequency(device, 0, zcl.AttributeDataTypeValue{
			DataType: 0,
			Value:    uint64(99),
		})

		assert.Equal(t, 49.5, i.data[device.Identifier].Mains[0].Frequency)
		assert.True(t, i.data[device.Identifier].LastChangeTime.Equal(currentTime) || i.data[device.Identifier].LastChangeTime.After(currentTime))
		assert.True(t, i.data[device.Identifier].LastUpdateTime.Equal(currentTime) || i.data[device.Identifier].LastUpdateTime.After(currentTime))
	})
}

func TestImplementation_attributeUpdateBatteryVoltage(t *testing.T) {
	t.Run("updating battery voltage via attribute updates data store", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   id,
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		i.data[device.Identifier] = Data{
			Battery: []*capabilities.PowerBatteryStatus{
				{
					Present: capabilities.Voltage,
				},
			},
		}

		currentTime := time.Now()

		i.attributeUpdateBatteryVoltage(device, 0, zcl.AttributeDataTypeValue{
			DataType: 0,
			Value:    uint64(34),
		})

		assert.Equal(t, 3.4, i.data[device.Identifier].Battery[0].Voltage)
		assert.True(t, i.data[device.Identifier].LastChangeTime.Equal(currentTime) || i.data[device.Identifier].LastChangeTime.After(currentTime))
		assert.True(t, i.data[device.Identifier].LastUpdateTime.Equal(currentTime) || i.data[device.Identifier].LastUpdateTime.After(currentTime))
	})
}

func TestImplementation_attributeUpdateBatteryPercentageRemaining(t *testing.T) {
	t.Run("updating battery percentage remaining via attribute updates data store", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   id,
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		i.data[device.Identifier] = Data{
			Battery: []*capabilities.PowerBatteryStatus{
				{
					Present: capabilities.Remaining,
				},
			},
		}

		currentTime := time.Now()

		i.attributeUpdateBatterPercentageRemaining(device, 0, zcl.AttributeDataTypeValue{
			DataType: 0,
			Value:    uint64(15),
		})

		assert.Equal(t, 0.075, i.data[device.Identifier].Battery[0].Remaining)
		assert.True(t, i.data[device.Identifier].LastChangeTime.Equal(currentTime) || i.data[device.Identifier].LastChangeTime.After(currentTime))
		assert.True(t, i.data[device.Identifier].LastUpdateTime.Equal(currentTime) || i.data[device.Identifier].LastUpdateTime.After(currentTime))
	})
}

func TestImplementation_attributeUpdateVendorXiaomiApproachOne(t *testing.T) {
	t.Run("updating battery voltage via attribute updates data store", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		i.supervisor = &zda.SimpleSupervisor{
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   id,
			Capabilities: []da.Capability{capabilities.PowerSupplyFlag},
		}

		i.data[device.Identifier] = Data{
			Battery: []*capabilities.PowerBatteryStatus{
				{
					Present: capabilities.Voltage,
				},
			},
		}

		currentTime := time.Now()

		i.attributeUpdateVendorXiaomiApproachOne(device, 0, zcl.AttributeDataTypeValue{
			DataType: 0,
			Value:    string([]byte{0x00, 0x20}),
		})

		assert.Equal(t, 3.2, i.data[device.Identifier].Battery[0].Voltage)
		assert.True(t, i.data[device.Identifier].LastChangeTime.Equal(currentTime) || i.data[device.Identifier].LastChangeTime.After(currentTime))
		assert.True(t, i.data[device.Identifier].LastUpdateTime.Equal(currentTime) || i.data[device.Identifier].LastUpdateTime.After(currentTime))
	})
}
