package on_off

import (
	"context"
	"fmt"
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
	t.Run("removes capability if no endpoints have the OnOff cluster", func(t *testing.T) {
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

		mockManageDeviceCapabilities.On("Remove", device, capabilities.OnOffFlag)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{
			State: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)

		assert.False(t, i.data[addr].State)
	})

	t.Run("adds capability and sets product data if on first endpoint that has OnOff cluster, successful bind and configure reporting", func(t *testing.T) {
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
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		mockZCL := mocks.MockZCL{}

		defer mockManageDeviceCapabilities.AssertExpectations(t)
		defer mockZCL.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Add", device, capabilities.OnOffFlag)
		mockZCL.On("Bind", mock.Anything, device, endpoint, zcl.OnOffId).Return(nil)
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			ZCLImpl:          &mockZCL,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.False(t, i.data[addr].RequiresPolling)
	})

	t.Run("adds capability and sets product data if on first endpoint that has OnOff cluster, failed bind, successful reporting", func(t *testing.T) {
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
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		mockZCL := mocks.MockZCL{}
		mockPoller := mocks.MockPoller{}

		defer mockManageDeviceCapabilities.AssertExpectations(t)
		defer mockZCL.AssertExpectations(t)
		defer mockPoller.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Add", device, capabilities.OnOffFlag)
		mockZCL.On("Bind", mock.Anything, device, endpoint, zcl.OnOffId).Return(fmt.Errorf("fail"))
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		ret := func() {}
		mockPoller.On("Add", device, 5*time.Second, mock.Anything).Return(ret)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			ZCLImpl:          &mockZCL,
			PollerImpl:       &mockPoller,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.True(t, i.data[addr].RequiresPolling)
		assert.NotNil(t, i.data[addr].PollerCancel)
	})

	t.Run("adds capability and sets product data if on first endpoint that has OnOff cluster, successful bind, failed reporting", func(t *testing.T) {
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
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
		}

		mockManageDeviceCapabilities := mocks.MockManageDeviceCapabilities{}
		mockZCL := mocks.MockZCL{}
		mockPoller := mocks.MockPoller{}

		defer mockManageDeviceCapabilities.AssertExpectations(t)
		defer mockZCL.AssertExpectations(t)
		defer mockPoller.AssertExpectations(t)

		mockManageDeviceCapabilities.On("Add", device, capabilities.OnOffFlag)
		mockZCL.On("Bind", mock.Anything, device, endpoint, zcl.OnOffId).Return(nil)
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, zcl.OnOffId, onoff.OnOff, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(fmt.Errorf("fail"))

		ret := func() {}
		mockPoller.On("Add", device, 5*time.Second, mock.Anything).Return(ret)

		i.supervisor = &zda.SimpleSupervisor{
			MDCImpl:          &mockManageDeviceCapabilities,
			ZCLImpl:          &mockZCL,
			PollerImpl:       &mockPoller,
			DeviceConfigImpl: &mocks.DefaultDeviceConfig{},
		}

		i.data[addr] = Data{}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.EnumerateDevice(ctx, device)
		assert.NoError(t, err)
		assert.Equal(t, zigbee.Endpoint(0x01), i.data[addr].Endpoint)
		assert.True(t, i.data[addr].RequiresPolling)
		assert.NotNil(t, i.data[addr].PollerCancel)
	})
}

func TestImplementation_zclCallback(t *testing.T) {
	t.Run("does nothing if the device matched does not have the capability", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State:           false,
				RequiresPolling: false,
				Endpoint:        0,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockDAES := mocks.MockDAEventSender{}

		i.supervisor = &zda.SimpleSupervisor{
			DAESImpl: &mockDAES,
		}

		mockDAES.AssertExpectations(t)

		endpoint := zigbee.Endpoint(0x01)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
		}

		i.zclCallback(device, zcl.Message{
			Command: &global.ReportAttributes{
				Records: []global.ReportAttributesRecord{
					{
						Identifier: onoff.OnOff,
						DataTypeValue: &zcl.AttributeDataTypeValue{
							DataType: zcl.TypeBoolean,
							Value:    true,
						},
					},
				},
			},
		})

		assert.False(t, i.data[addr].State)
	})

	t.Run("sets new state if device does have the capability", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State:           false,
				RequiresPolling: false,
				Endpoint:        0,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockDAES := mocks.MockDAEventSender{}
		mockCDAD := mocks.MockComposeDADevice{}
		defer mockDAES.AssertExpectations(t)
		defer mockCDAD.AssertExpectations(t)

		i.supervisor = &zda.SimpleSupervisor{
			DAESImpl: &mockDAES,
			CDADImpl: &mockCDAD,
		}

		endpoint := zigbee.Endpoint(0x01)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{capabilities.OnOffFlag},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.OnOffId},
				},
			},
		}

		daDevice := da.BaseDevice{}
		mockCDAD.On("Compose", device).Return(daDevice)
		mockDAES.On("Send", capabilities.OnOffState{
			Device: daDevice,
			State:  true,
		})

		i.zclCallback(device, zcl.Message{
			Command: &global.ReportAttributes{
				Records: []global.ReportAttributesRecord{
					{
						Identifier: onoff.OnOff,
						DataTypeValue: &zcl.AttributeDataTypeValue{
							DataType: zcl.TypeBoolean,
							Value:    true,
						},
					},
				},
			},
		})

		assert.True(t, i.data[addr].State)
	})
}
