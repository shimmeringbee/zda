package zda

import (
	"context"
	. "github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

var testGatewayIEEEAddress = zigbee.IEEEAddress(0x0102030405060708)
var testGatewayNetworkAddress = zigbee.NetworkAddress(0xeeff)

func NewTestZigbeeGateway() (*ZigbeeGateway, *zigbee.MockProvider, func(*testing.T)) {
	mockProvider := new(zigbee.MockProvider)

	mockProvider.On("AdapterNode").Return(zigbee.Node{
		IEEEAddress:    testGatewayIEEEAddress,
		NetworkAddress: testGatewayNetworkAddress,
	})
	zgw := New(mockProvider)

	return zgw, mockProvider, func(t *testing.T) {
		zgw.Stop()
		mockProvider.AssertExpectations(t)
	}
}

func TestZigbeeGateway_Contract(t *testing.T) {
	t.Run("can be assigned to a da.Gateway", func(t *testing.T) {
		assert.Implements(t, (*Gateway)(nil), new(ZigbeeGateway))
	})
}

func TestZigbeeGateway_New(t *testing.T) {
	t.Run("a new gateway that is configured and started, has a self device which is valid", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		expectedDevice := Device{
			Gateway:    zgw,
			Identifier: testGatewayIEEEAddress,
			Capabilities: []Capability{
				DeviceDiscoveryFlag,
			},
		}

		actualDevice := zgw.Self()

		assert.Equal(t, expectedDevice, actualDevice)
	})
}

func TestZigbeeGateway_Devices(t *testing.T) {
	t.Run("devices returns self", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		expectedDevice := Device{
			Gateway:    zgw,
			Identifier: testGatewayIEEEAddress,
			Capabilities: []Capability{
				DeviceDiscoveryFlag,
			},
		}

		expectedDevices := []Device{expectedDevice}
		actualDevices := zgw.Devices()

		assert.Equal(t, expectedDevices, actualDevices)
	})
}

func TestZigbeeGateway_ReadEvent(t *testing.T) {
	t.Run("context which expires should result in error", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := zgw.ReadEvent(ctx)
		assert.Error(t, err)
	})

	t.Run("sent events are received through ReadEvent", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedEvent := true

		go func() {
			zgw.sendEvent(expectedEvent)
		}()

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)
	})
}

func TestZigbeeGateway_DeviceAdded(t *testing.T) {
	t.Run("a DeviceAdded event is sent when a Zigbee device is announced by the provider, is placed in the store and calls internal callbacks", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, mock.Anything).Maybe().Return(zigbee.NodeDescription{}, nil)
		mockProvider.On("QueryNodeEndpoints", mock.Anything, mock.Anything).Maybe().Return([]zigbee.Endpoint{}, nil)
		zgw.Start()
		defer stop(t)

		callbackCalled := false

		zgw.callbacks.Add(func(ctx context.Context, event internalNodeJoin) error {
			callbackCalled = true
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedAddress := zigbee.IEEEAddress(0x0102030405060708)

		mockCall.RunFn = multipleReadEvents(mockCall, zigbee.NodeJoinEvent{
			Node: zigbee.Node{
				IEEEAddress:    expectedAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, nil)

		expectedEvent := DeviceAdded{
			Device: Device{
				Gateway:      zgw,
				Identifier:   expectedAddress,
				Capabilities: []Capability{EnumerateDeviceFlag},
			},
		}

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		_, found := zgw.getDevice(expectedAddress)
		assert.True(t, found)
	})

	t.Run("only one DeviceAdded event is sent when a Zigbee device is announced by the provider twice", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, mock.Anything).Maybe().Return(zigbee.NodeDescription{}, nil)
		mockProvider.On("QueryNodeEndpoints", mock.Anything, mock.Anything).Maybe().Return([]zigbee.Endpoint{}, nil)

		zgw.Start()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedAddress := zigbee.IEEEAddress(0x0102030405060708)

		mockCall.RunFn = multipleReadEvents(mockCall, zigbee.NodeJoinEvent{
			Node: zigbee.Node{
				IEEEAddress:    expectedAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, zigbee.NodeJoinEvent{
			Node: zigbee.Node{
				IEEEAddress:    expectedAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, nil)

		expectedEvent := DeviceAdded{
			Device: Device{
				Gateway:      zgw,
				Identifier:   expectedAddress,
				Capabilities: []Capability{EnumerateDeviceFlag},
			},
		}

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)
	})
}

func TestZigbeeGateway_DeviceRemoved(t *testing.T) {
	t.Run("a DeviceRemoved event is sent when a Zigbee device is removed by the provider and is delete from the store", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		zgw.Start()
		defer stop(t)

		callbackCalled := false

		zgw.callbacks.Add(func(ctx context.Context, event internalNodeLeave) error {
			callbackCalled = true
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedAddress := zigbee.IEEEAddress(0x0102030405060708)
		zgw.addDevice(expectedAddress)

		mockCall.RunFn = multipleReadEvents(mockCall, zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress:    expectedAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, nil)

		expectedEvent := DeviceRemoved{
			Device: Device{
				Gateway:      zgw,
				Identifier:   expectedAddress,
				Capabilities: []Capability{EnumerateDeviceFlag},
			},
		}

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		_, found := zgw.getDevice(expectedAddress)
		assert.False(t, found)
	})

	t.Run("a DeviceRemoved event is not sent when a Zigbee device is removed by the provider but is not in the device store", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		zgw.Start()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedAddress := zigbee.IEEEAddress(0x0102030405060708)

		mockCall.RunFn = multipleReadEvents(mockCall, zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress:    expectedAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, nil)

		_, err := zgw.ReadEvent(ctx)
		assert.Error(t, err)
	})
}

func TestZigbeeGateway_DeviceStore(t *testing.T) {
	t.Run("device store performs basic actions", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		id := zigbee.IEEEAddress(0x0102030405060708)

		_, found := zgw.getDevice(id)
		assert.False(t, found)

		zDev := zgw.addDevice(id)
		assert.Equal(t, id, zDev.device.Identifier)
		assert.Equal(t, zgw, zDev.device.Gateway)
		assert.Equal(t, []Capability{EnumerateDeviceFlag}, zDev.device.Capabilities)

		zDev, found = zgw.getDevice(id)
		assert.True(t, found)
		assert.Equal(t, id, zDev.device.Identifier)

		zgw.removeDevice(id)

		_, found = zgw.getDevice(id)
		assert.False(t, found)
	})
}

func multipleReadEvents(call *mock.Call, events ...interface{}) func(mock.Arguments) {
	var count int

	return func(arguments mock.Arguments) {
		var event interface{}

		if count >= len(events) {
			event = events[len(events)-1]
		} else {
			event = events[count]
		}

		call.ReturnArguments = mock.Arguments{event, nil}
		count += 1
	}
}
