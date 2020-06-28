package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sort"
	"testing"
	"time"
)

func TestZigbeeEnumerateCapabilities_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.EnumerateDevice", func(t *testing.T) {
		assert.Implements(t, (*EnumerateDevice)(nil), new(ZigbeeEnumerateDevice))
	})
}

func TestZigbeeGateway_ReturnsEnumerateCapabilitiesCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(EnumerateDeviceFlag)
		assert.IsType(t, (*ZigbeeEnumerateDevice)(nil), actualZdd)
	})
}

func TestZigbeeEnumerateCapabilities_Enumerate(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		nonSelfDevice := da.Device{}

		err := zed.Enumerate(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		nonCapability := da.Device{Gateway: zgw}

		err := zed.Enumerate(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("queues a device for enumeration", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		ieeeAddress := zigbee.IEEEAddress(0x01)
		iNode := zgw.addNode(ieeeAddress)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x00}
		iDev := zgw.addDevice(subId, iNode)

		// Stop the worker routines so that we can examine the queue, with 50ms cooldown to allow end.
		zed.Stop()
		time.Sleep(50 * time.Millisecond)

		err := zed.Enumerate(context.Background(), iDev.device)
		assert.NoError(t, err)

		select {
		case qNode := <-zed.queue:
			assert.Equal(t, iNode, qNode)
		default:
			assert.Fail(t, "no iDev was queued")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		startEvent := event.(EnumerateDeviceStart)
		assert.Equal(t, iDev.device, startEvent.Device)
	})

	t.Run("queues a device for enumeration on internal join message", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		ieeeAddress := zigbee.IEEEAddress(0x01)
		iNode := zgw.addNode(ieeeAddress)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: ieeeAddress, SubIdentifier: 0x00}
		iDev := zgw.addDevice(subId, iNode)

		// Stop the worker routines so that we can examine the queue, with 50ms cooldown to allow end.
		zed.Stop()
		time.Sleep(50 * time.Millisecond)

		err := zgw.callbacks.Call(context.Background(), internalNodeJoin{node: iNode})
		assert.NoError(t, err)

		select {
		case qNode := <-zed.queue:
			assert.Equal(t, iNode, qNode)
		default:
			assert.Fail(t, "no iDev was queued")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		startEvent := event.(EnumerateDeviceStart)
		assert.Equal(t, iDev.device, startEvent.Device)
	})
}

func TestZigbeeEnumerateDevice_enumerateDevice(t *testing.T) {
	t.Run("enumerating a device requests a node description and endpoints", func(t *testing.T) {
		expectedIEEE := zigbee.IEEEAddress(0x00112233445566)
		expectedNodeDescription := zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		}
		expectedEndpoints := []zigbee.Endpoint{0x01, 0x02}

		expectedEndpointDescs := []zigbee.EndpointDescription{
			{
				Endpoint:       0x01,
				ProfileID:      0x02,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  nil,
				OutClusterList: nil,
			},
			{
				Endpoint:       0x02,
				ProfileID:      0x03,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  nil,
				OutClusterList: nil,
			},
		}

		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(expectedNodeDescription, nil)
		mockProvider.On("QueryNodeEndpoints", mock.Anything, expectedIEEE).Return(expectedEndpoints, nil)
		mockProvider.On("QueryNodeEndpointDescription", mock.Anything, expectedIEEE, zigbee.Endpoint(0x01)).Return(expectedEndpointDescs[0], nil)
		mockProvider.On("QueryNodeEndpointDescription", mock.Anything, expectedIEEE, zigbee.Endpoint(0x02)).Return(expectedEndpointDescs[1], nil)
		zgw.Start()
		defer stop(t)

		callbackCalled := false

		zgw.callbacks.Add(func(ctx context.Context, event internalNodeEnumeration) error {
			callbackCalled = true
			return nil
		})

		iNode := zgw.addNode(expectedIEEE)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedIEEE, SubIdentifier: 0x00}
		iDev := zgw.addDevice(subId, iNode)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		err := zed.Enumerate(context.Background(), iDev.device)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		successEvent := event.(EnumerateDeviceSuccess)
		assert.Equal(t, subId, successEvent.Device.Identifier)

		assert.Equal(t, expectedNodeDescription, iNode.nodeDesc)
		assert.Equal(t, expectedEndpoints, iNode.endpoints)
		assert.Equal(t, expectedEndpointDescs[0], iNode.endpointDescriptions[0x01])
		assert.Equal(t, expectedEndpointDescs[1], iNode.endpointDescriptions[0x02])

		assert.True(t, callbackCalled)
	})

	t.Run("enumerating a device handles a failure during QueryNodeDescription", func(t *testing.T) {
		expectedError := errors.New("expected error")
		expectedIEEE := zigbee.IEEEAddress(0x00112233445566)

		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(zigbee.NodeDescription{}, expectedError)
		zgw.Start()
		defer stop(t)

		iNode := zgw.addNode(expectedIEEE)
		subId := IEEEAddressWithSubIdentifier{IEEEAddress: expectedIEEE, SubIdentifier: 0x00}
		iDev := zgw.addDevice(subId, iNode)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		err := zed.Enumerate(context.Background(), iDev.device)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		failureEvent := event.(EnumerateDeviceFailure)
		assert.Equal(t, subId, failureEvent.Device.Identifier)
		assert.Equal(t, expectedError, failureEvent.Error)
	})
}

func TestZigbeeEnumerateDevice_allocateEndpointsToDevices(t *testing.T) {
	t.Run("allocating endpoints to devices results in endpoints with same device ID being mapped to the same internalDevice", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)
		subIdZero := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 0x00}
		subIdOne := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 0x01}
		zgw.addDevice(subIdZero, iNode)

		iNode.endpoints = []zigbee.Endpoint{0x10, 0x20, 0x11}
		iNode.endpointDescriptions[0x10] = zigbee.EndpointDescription{
			Endpoint:       0x10,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x10,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		iNode.endpointDescriptions[0x11] = zigbee.EndpointDescription{
			Endpoint:       0x11,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x10,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		iNode.endpointDescriptions[0x20] = zigbee.EndpointDescription{
			Endpoint:       0x20,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x20,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		zed.allocateEndpointsToDevices(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, iNode.devices[subIdZero].endpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)
	})

	t.Run("executing allocating endpoints twice does not result in duplicate endpoints", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)
		subIdZero := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 0x00}
		subIdOne := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 0x01}
		zgw.addDevice(subIdZero, iNode)

		iNode.endpoints = []zigbee.Endpoint{0x10, 0x20, 0x11}
		iNode.endpointDescriptions[0x10] = zigbee.EndpointDescription{
			Endpoint:       0x10,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x10,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		iNode.endpointDescriptions[0x11] = zigbee.EndpointDescription{
			Endpoint:       0x11,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x10,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		iNode.endpointDescriptions[0x20] = zigbee.EndpointDescription{
			Endpoint:       0x20,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       0x20,
			DeviceVersion:  1,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		zed.allocateEndpointsToDevices(iNode)
		zed.allocateEndpointsToDevices(iNode)

		sortedEndpoints := iNode.devices[subIdZero].endpoints

		sort.SliceStable(sortedEndpoints, func(i, j int) bool {
			return sortedEndpoints[i] < sortedEndpoints[j]
		})

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, sortedEndpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)
	})
}

func TestZigbeeEnumerateDevice_removeMissingEndpointDescriptions(t *testing.T) {
	t.Run("removes endpoint descriptions of the store if the endpoints are not in the endpoints list", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{}

		zed.removeMissingEndpointDescriptions(iNode)

		_, found := iNode.endpointDescriptions[0x02]
		assert.False(t, found)
	})
}

func TestZigbeeEnumerateDevice_deallocateDevicesFromMissingEndpoints(t *testing.T) {
	t.Run("removes endpoint from a device which no longer matches", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)

		iDevOneId := iNode.findNextDeviceIdentifier()
		iDevOne := zgw.addDevice(iDevOneId, iNode)

		iDevTwoId := iNode.findNextDeviceIdentifier()
		iDevTwo := zgw.addDevice(iDevTwoId, iNode)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x02}

		iDevOne.deviceID = 0x01
		iDevOne.endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevTwo.deviceID = 0x02
		iDevTwo.endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01}, iDevOne.endpoints)
		assert.Equal(t, []zigbee.Endpoint{0x02}, iDevTwo.endpoints)
	})

	t.Run("removes device which has had all endpoints removed", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)

		iDevOneId := iNode.findNextDeviceIdentifier()
		iDevOne := zgw.addDevice(iDevOneId, iNode)

		iDevTwoId := iNode.findNextDeviceIdentifier()
		iDevTwo := zgw.addDevice(iDevTwoId, iNode)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevOne.deviceID = 0x01
		iDevOne.endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevTwo.deviceID = 0x02
		iDevTwo.endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01, 0x02}, iDevOne.endpoints)

		_, found := zgw.getDevice(iDevTwoId)
		assert.False(t, found)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, err := zgw.ReadEvent(ctx)

		assert.NoError(t, err)
		assert.IsType(t, da.DeviceRemoved{}, event)
	})

	t.Run("does not remove sole remaining device from a node", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)

		ieee := zigbee.IEEEAddress(0x00112233445566)
		iNode := zgw.addNode(ieee)

		iDevOneId := iNode.findNextDeviceIdentifier()
		iDevOne := zgw.addDevice(iDevOneId, iNode)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevOne.deviceID = 0x02
		iDevOne.endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{}, iDevOne.endpoints)

		_, found := zgw.getDevice(iDevOneId)
		assert.True(t, found)
	})
}
