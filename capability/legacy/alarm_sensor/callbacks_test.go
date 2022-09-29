package alarm_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/ias_zone"
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
	t.Run("removes capability if no endpoints have the AlarmSensorFlag cluster", func(t *testing.T) {
		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				0x00: {
					Endpoint:      0x00,
					InClusterList: []zigbee.ClusterID{},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Remove", device, capabilities.AlarmSensorFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{
			Alarms: map[capabilities.SensorType]bool{capabilities.Radiation: false},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Empty(t, i.data[addr].Alarms)
	})

	t.Run("adds capability and sets product data if on first endpoint that has AlarmSensorFlag cluster", func(t *testing.T) {
		coordIEEE := zigbee.GenerateLocalAdministeredIEEEAddress()

		mockDL := mocks.MockDeviceLookup{}
		defer mockDL.AssertExpectations(t)
		mockDL.On("Self").Return(zda.Device{
			Identifier: zda.IEEEAddressWithSubIdentifier{IEEEAddress: coordIEEE},
		})

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
					InClusterList: []zigbee.ClusterID{zcl.IASZoneId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.AlarmSensorFlag)

		mockZCL.On("WriteAttributes", mock.Anything, device, endpoint, zcl.IASZoneId, map[zcl.AttributeID]zcl.AttributeDataTypeValue{ias_zone.IASCIEAddress: {DataType: zcl.TypeIEEEAddress, Value: coordIEEE}}).
			Return(map[zcl.AttributeID]global.WriteAttributesResponseRecord{ias_zone.IASCIEAddress: {Status: 0, Identifier: ias_zone.IASCIEAddress}}, nil)

		mockZCL.On("WaitForMessage", mock.Anything, device, endpoint, zcl.IASZoneId, ias_zone.ZoneEnrollRequestId).Return(zcl.Message{
			Command: &ias_zone.ZoneEnrollRequest{
				ZoneType:         0x0001,
				ManufacturerCode: 0,
			},
		}, nil)

		mockZCL.On("SendCommand", mock.Anything, device, endpoint, zcl.IASZoneId, &ias_zone.ZoneEnrollResponse{}).Return(nil)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.IASZoneId, []zcl.AttributeID{ias_zone.ZoneState, ias_zone.ZoneStatus}).Return(map[zcl.AttributeID]global.ReadAttributeResponseRecord{
			ias_zone.ZoneState: {
				Identifier: ias_zone.ZoneState,
				Status:     0,
				DataTypeValue: &zcl.AttributeDataTypeValue{
					DataType: zcl.TypeEnum8,
					Value:    uint8(1),
				},
			},
			ias_zone.ZoneStatus: {
				Identifier: ias_zone.ZoneStatus,
				Status:     0,
				DataTypeValue: &zcl.AttributeDataTypeValue{
					DataType: zcl.TypeBitmap16,
					Value:    uint64(0xffff),
				},
			},
		}, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			ZCLImpl:          &mockZCL,
			DLImpl:           &mockDL,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		i.data[addr] = Data{}

		expectedSensorStates := map[capabilities.SensorType]bool{
			capabilities.SecurityInfrastructure: true,
			capabilities.DeviceTamper:           true,
			capabilities.DeviceBatteryLow:       true,
			capabilities.DeviceBatteryFailure:   true,
			capabilities.DeviceMainsFailure:     true,
			capabilities.DeviceFailure:          true,
			capabilities.DeviceTest:             true,
			0xffff:                              true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.Equal(t, uint16(0x0001), i.data[addr].ZoneType)
		assert.Equal(t, expectedSensorStates, i.data[addr].Alarms)
	})
}

func TestImplementation_zoneStatusChangeNotification(t *testing.T) {
	t.Run("updates alarm states and sends event", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{capabilities.AlarmSensorFlag},
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		mockEventSender := &mocks.MockDAEventSender{}
		defer mockEventSender.AssertExpectations(t)

		i.supervisor = &zda.SimpleSupervisor{
			CDADImpl:         &zda.ComposeDADeviceShim{},
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			DAESImpl:         mockEventSender,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		mockEventSender.On("Send", capabilities.AlarmSensorUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			States: map[capabilities.SensorType]bool{
				capabilities.FireOther:            true,
				capabilities.DeviceTamper:         true,
				capabilities.DeviceBatteryLow:     true,
				capabilities.DeviceBatteryFailure: true,
				capabilities.DeviceMainsFailure:   true,
				capabilities.DeviceFailure:        true,
				capabilities.DeviceTest:           true,
				0xffff:                            true},
		})

		i.data[addr] = Data{ZoneType: 0x0028}

		currentTime := time.Now()

		i.zoneStatusChangeNotification(device, zcl.Message{Command: &ias_zone.ZoneStatusChangeNotification{
			Reserved:           0,
			BatteryDefect:      true,
			TestMode:           true,
			ACMainsFault:       true,
			Trouble:            true,
			RestoreReports:     true,
			SupervisionReports: true,
			BatteryLow:         true,
			Tamper:             true,
			Alarm2:             true,
			Alarm1:             true,
			ExtendedStatus:     0,
			ZoneID:             0,
			Delay:              0,
		}})

		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})
}
