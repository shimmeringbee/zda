package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
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
		mockAM := mocks.MockAttributeMonitor{}
		defer mockAM.AssertExpectations(t)

		i := &Implementation{attMonTemperatureMeasurementCluster: &mockAM, attMonVendorXiaomiApproachOne: &mockAM}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
		i.datalock = &sync.RWMutex{}

		id := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := zda.Device{
			Identifier: id,
		}

		mockAM.On("Detach", mock.Anything, device).Twice()

		i.data[id] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.RemovedDevice(ctx, device)

		assert.NoError(t, err)
		assert.NotContains(t, i.data, id)
	})
}

func TestImplementation_enumerateDeviceCallback(t *testing.T) {
	t.Run("removes capability if no endpoints have the TemperatureSensor cluster", func(t *testing.T) {
		mockAM := mocks.MockAttributeMonitor{}
		defer mockAM.AssertExpectations(t)

		i := &Implementation{attMonTemperatureMeasurementCluster: &mockAM, attMonVendorXiaomiApproachOne: &mockAM}
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

		mockAM.On("Detach", mock.Anything, device).Twice()

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Remove", device, capabilities.TemperatureSensorFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{
			State: 1,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.Equal(t, float64(0), i.data[addr].State)
	})

	t.Run("adds capability and sets product data if on first endpoint that has TemperatureSensor cluster, puts requires polling in data", func(t *testing.T) {
		mockAM := mocks.MockAttributeMonitor{}
		defer mockAM.AssertExpectations(t)

		i := &Implementation{attMonTemperatureMeasurementCluster: &mockAM}
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
					InClusterList: []zigbee.ClusterID{zcl.TemperatureMeasurementId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.TemperatureSensorFlag)

		mockAM.On("Attach", mock.Anything, device, endpoint, 0).Return(true, nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.True(t, i.data[addr].RequiresPolling)
	})

	t.Run("adds capability from Xaiomi vendor specific attribute if available", func(t *testing.T) {
		mockAMZCLCluster := mocks.MockAttributeMonitor{}
		defer mockAMZCLCluster.AssertExpectations(t)

		mockAMXiaomi := mocks.MockAttributeMonitor{}
		defer mockAMXiaomi.AssertExpectations(t)

		i := &Implementation{attMonTemperatureMeasurementCluster: &mockAMZCLCluster, attMonVendorXiaomiApproachOne: &mockAMXiaomi}
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
					InClusterList: []zigbee.ClusterID{zcl.TemperatureMeasurementId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		defer mockManageDeviceCapabilities.AssertExpectations(t)
		mockManageDeviceCapabilities.On("Add", device, capabilities.TemperatureSensorFlag)

		mockAMXiaomi.On("Attach", mock.Anything, device, endpoint, nil).Return(true, nil)

		mockConfig := mocks.MockConfig{}
		defer mockConfig.AssertExpectations(t)
		mockConfig.On("Bool", "HasTemperatureMeasurementZCLCluster", mock.Anything).Return(false)
		mockConfig.On("Bool", "HasVendorXiaomiApproachOne", mock.Anything).Return(true)
		mockConfig.On("Int", "BasicEndpoint", mock.Anything).Return(1)

		mockDeviceConfig := mocks.MockDeviceConfig{}
		defer mockDeviceConfig.AssertExpectations(t)
		mockDeviceConfig.On("Get", device, "TemperatureSensor").Return(&mockConfig)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mockDeviceConfig,
			LoggerImpl:       logwrap.New(discard.Discard()),
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.True(t, i.data[addr].RequiresPolling)
		assert.True(t, i.data[addr].VendorXiaomiApproachOne)
	})
}
