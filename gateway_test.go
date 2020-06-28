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
	t.Run("a new gateway that is configured and started, has a self device which is valid and has registered all standard profiles", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()

		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, zigbee.Endpoint(1), zigbee.ProfileHomeAutomation, uint16(1), uint8(1), []zigbee.ClusterID{}, []zigbee.ClusterID{}).Return(nil)

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

		mockProvider.AssertExpectations(t)
	})
}

func TestZigbeeGateway_Devices(t *testing.T) {
	t.Run("devices returns self", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		expectedDeviceId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedAddress, SubIdentifier: 0x00}

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
				Identifier:   expectedDeviceId,
				Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
			},
		}

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		node, found := zgw.getNode(expectedAddress)
		assert.True(t, found)

		assert.Equal(t, node.ieeeAddress, expectedAddress)
		_, deviceFound := node.devices[expectedDeviceId]
		assert.True(t, deviceFound)

		device, found := zgw.getDevice(expectedDeviceId)
		assert.True(t, found)
		assert.Equal(t, node, device.node)
	})

	t.Run("only one DeviceAdded event is sent when a Zigbee device is announced by the provider twice", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, mock.Anything).Maybe().Return(zigbee.NodeDescription{}, nil)
		mockProvider.On("QueryNodeEndpoints", mock.Anything, mock.Anything).Maybe().Return([]zigbee.Endpoint{}, nil)

		zgw.Start()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedAddress := zigbee.IEEEAddress(0x0102030405060708)
		expectedDeviceId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedAddress, SubIdentifier: 0x00}

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
				Identifier:   expectedDeviceId,
				Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
			},
		}

		actualEventOne, err := zgw.ReadEvent(ctx)
		assert.IsType(t, DeviceAdded{}, actualEventOne)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEventOne)

		actualEventTwo, err := zgw.ReadEvent(ctx)
		assert.NotNil(t, actualEventTwo)
		assert.NoError(t, err)
		assert.IsType(t, EnumerateDeviceStart{}, actualEventTwo)

		actualEventThree, err := zgw.ReadEvent(ctx)
		assert.NotNil(t, actualEventThree)
		assert.NoError(t, err)
		assert.IsType(t, EnumerateDeviceSuccess{}, actualEventThree)
	})
}

func TestZigbeeGateway_DeviceRemoved(t *testing.T) {
	t.Run("a DeviceRemoved event is sent when a Zigbee device is removed by the provider and is delete from the store", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		node := zgw.addNode(expectedAddress)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedAddress, SubIdentifier: 0x00}
		zgw.addDevice(subId, node)

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
				Identifier:   subId,
				Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
			},
		}

		_, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		_, found := zgw.getDevice(subId)
		assert.False(t, found)

		_, found = zgw.getNode(expectedAddress)
		assert.False(t, found)
	})

	t.Run("a DeviceRemoved event is sent for each device on a Zigbee node when it is is removed by the provider and is delete from the store", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		callbackCalled := false

		zgw.callbacks.Add(func(ctx context.Context, event internalNodeLeave) error {
			callbackCalled = true
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		subIdOne := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x01}
		subIdTwo := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x02}

		node := zgw.addNode(ieeeAddress)
		zgw.addDevice(subIdOne, node)
		zgw.addDevice(subIdTwo, node)

		mockCall.RunFn = multipleReadEvents(mockCall, zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress:    ieeeAddress,
				NetworkAddress: 0,
				LogicalType:    0,
				LQI:            0,
				Depth:          0,
				LastDiscovered: time.Time{},
				LastReceived:   time.Time{},
			},
		}, nil)

		expectedEvent := []DeviceRemoved{
			{
				Device: Device{
					Gateway:      zgw,
					Identifier:   subIdOne,
					Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
				},
			},
			{
				Device: Device{
					Gateway:      zgw,
					Identifier:   subIdTwo,
					Capabilities: []Capability{EnumerateDeviceFlag, LocalDebugFlag},
				},
			},
		}

		_, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent[0], actualEvent)

		actualEvent, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent[1], actualEvent)

		assert.True(t, callbackCalled)

		_, found := zgw.getDevice(subIdOne)
		assert.False(t, found)

		_, found = zgw.getDevice(subIdTwo)
		assert.False(t, found)

		_, found = zgw.getNode(ieeeAddress)
		assert.False(t, found)
	})

	t.Run("a DeviceRemoved event is not sent when a Zigbee device is removed by the provider but is not in the device store", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockCall := mockProvider.On("ReadEvent", mock.Anything).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
		count++
	}
}
