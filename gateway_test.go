package zda

import (
	"context"
	"github.com/shimmeringbee/da"
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
	t.Run("can be assigned to a da.gateway", func(t *testing.T) {
		assert.Implements(t, (*da.Gateway)(nil), new(ZigbeeGateway))
	})
}

func TestZigbeeGateway_New(t *testing.T) {
	t.Run("a new gateway that is configured and started, has a self device which is valid and has registered all standard profiles", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()

		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, zigbee.Endpoint(1), zigbee.ProfileHomeAutomation, uint16(1), uint8(1), []zigbee.ClusterID{}, []zigbee.ClusterID{}).Return(nil)

		zgw.Start()
		defer stop(t)

		expectedDevice := da.BaseDevice{
			DeviceGateway:    zgw,
			DeviceIdentifier: IEEEAddressWithSubIdentifier{IEEEAddress: testGatewayIEEEAddress, SubIdentifier: 0},
			DeviceCapabilities: []da.Capability{
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

		expectedDevice := da.BaseDevice{
			DeviceGateway:    zgw,
			DeviceIdentifier: IEEEAddressWithSubIdentifier{IEEEAddress: testGatewayIEEEAddress, SubIdentifier: 0},
			DeviceCapabilities: []da.Capability{
				DeviceDiscoveryFlag,
			},
		}

		expectedDevices := []da.Device{expectedDevice}
		actualDevices := zgw.Devices()

		assert.Equal(t, expectedDevices, actualDevices)
	})

	t.Run("devices returns self and any other devices on gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieee := zigbee.IEEEAddress(0x01)
		sub := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 0}

		zgw.nodeTable.createNode(ieee)
		iDev, _ := zgw.nodeTable.createDevice(sub)

		expectedDevice := da.BaseDevice{
			DeviceGateway:    zgw,
			DeviceIdentifier: IEEEAddressWithSubIdentifier{IEEEAddress: testGatewayIEEEAddress, SubIdentifier: 0},
			DeviceCapabilities: []da.Capability{
				DeviceDiscoveryFlag,
			},
		}

		expectedDevices := []da.Device{expectedDevice, iDev.toDevice(zgw)}
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
	t.Run("a DeviceAdded event is sent when a Zigbee device is announced by the provider, is placed in the store and calls internal internalCallbacks", func(t *testing.T) {
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

		expectedEvent := da.DeviceAdded{
			Device: da.BaseDevice{
				DeviceGateway:      zgw,
				DeviceIdentifier:   expectedDeviceId,
				DeviceCapabilities: []da.Capability{EnumerateDeviceFlag},
			},
		}

		time.Sleep(50 * time.Millisecond)

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		node := zgw.nodeTable.getNode(expectedAddress)
		assert.NotNil(t, node)

		assert.Equal(t, node.ieeeAddress, expectedAddress)
		_, deviceFound := node.devices[expectedDeviceId.SubIdentifier]
		assert.True(t, deviceFound)

		dev := zgw.nodeTable.getDevice(expectedDeviceId)
		assert.NotNil(t, dev)
		assert.Equal(t, node, dev.node)
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

		expectedEvent := da.DeviceAdded{
			Device: da.BaseDevice{
				DeviceGateway:      zgw,
				DeviceIdentifier:   expectedDeviceId,
				DeviceCapabilities: []da.Capability{EnumerateDeviceFlag},
			},
		}

		actualEventOne, err := zgw.ReadEvent(ctx)
		assert.IsType(t, da.DeviceAdded{}, actualEventOne)
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
		zgw.nodeTable.createNode(expectedAddress)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedAddress, SubIdentifier: 0x00}
		zgw.nodeTable.createDevice(subId)

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

		expectedEvent := da.DeviceRemoved{
			Device: da.BaseDevice{
				DeviceGateway:      zgw,
				DeviceIdentifier:   subId,
				DeviceCapabilities: []da.Capability{EnumerateDeviceFlag},
			},
		}

		_, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)

		assert.True(t, callbackCalled)

		dev := zgw.nodeTable.getDevice(subId)
		assert.Nil(t, dev)

		node := zgw.nodeTable.getNode(expectedAddress)
		assert.Nil(t, node)
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

		zgw.nodeTable.createNode(ieeeAddress)
		zgw.nodeTable.createDevice(subIdOne)
		zgw.nodeTable.createDevice(subIdTwo)

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

		expectedEvent := []da.DeviceRemoved{
			{
				Device: da.BaseDevice{
					DeviceGateway:      zgw,
					DeviceIdentifier:   subIdOne,
					DeviceCapabilities: []da.Capability{EnumerateDeviceFlag},
				},
			},
			{
				Device: da.BaseDevice{
					DeviceGateway:      zgw,
					DeviceIdentifier:   subIdTwo,
					DeviceCapabilities: []da.Capability{EnumerateDeviceFlag},
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

		dev := zgw.nodeTable.getDevice(subIdOne)
		assert.Nil(t, dev)

		dev = zgw.nodeTable.getDevice(subIdTwo)
		assert.Nil(t, dev)

		node := zgw.nodeTable.getNode(ieeeAddress)
		assert.Nil(t, node)
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
