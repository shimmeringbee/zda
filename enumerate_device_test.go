package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestZigbeeEnumerateCapabilities_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.EnumerateDevice", func(t *testing.T) {
		assert.Implements(t, (*EnumerateDevice)(nil), new(ZigbeeEnumerateDevice))
	})
}

func TestZigbeeEnumerateCapabilities_Init(t *testing.T) {
	t.Run("registers against callbacks", func(t *testing.T) {
		mockAdderCaller := mockAdderCaller{}

		mockAdderCaller.On("Add", mock.AnythingOfType("func(context.Context, zda.internalNodeJoin) error"))

		zed := ZigbeeEnumerateDevice{
			internalCallbacks: &mockAdderCaller,
		}

		zed.Init()

		mockAdderCaller.AssertExpectations(t)
	})
}

func TestZigbeeEnumerateCapabilities_Enumerate(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zed := ZigbeeEnumerateDevice{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		err := zed.Enumerate(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zed := ZigbeeEnumerateDevice{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zed.gateway}

		err := zed.Enumerate(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("queues a device for enumeration", func(t *testing.T) {
		mockGateway := mockGateway{}

		iNode, iDev := generateTestNodeAndDevice()
		iNode.gateway = &mockGateway
		iDev.capabilities = []da.Capability{EnumerateDeviceFlag}

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("getDevice", iDev.identifier).Return(iDev, true)

		expectedEvent := EnumerateDeviceStart{
			Device: iDev.toDevice(),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedEvent)

		zed := ZigbeeEnumerateDevice{
			gateway:     &mockGateway,
			deviceStore: &mockDeviceStore,
			eventSender: &mockEventSender,
		}

		// Stop and start zed to populate queues
		zed.Start()
		zed.Stop()
		time.Sleep(time.Millisecond)

		err := zed.Enumerate(context.Background(), iDev.toDevice())
		assert.NoError(t, err)

		select {
		case qNode := <-zed.queue:
			assert.Equal(t, iNode, qNode)
		default:
			assert.Fail(t, "no iDev was queued")
		}

		mockDeviceStore.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
	})

	t.Run("queues a device for enumeration on internal join message", func(t *testing.T) {
		mockGateway := mockGateway{}

		iNode, iDev := generateTestNodeAndDevice()

		expectedEvent := EnumerateDeviceStart{
			Device: iDev.toDevice(),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedEvent)

		zed := ZigbeeEnumerateDevice{
			gateway:     &mockGateway,
			eventSender: &mockEventSender,
		}

		// Stop and start zed to populate queues
		zed.Start()
		zed.Stop()
		time.Sleep(time.Millisecond)

		err := zed.NodeJoinCallback(context.Background(), internalNodeJoin{node: iNode})
		assert.NoError(t, err)

		select {
		case qNode := <-zed.queue:
			assert.Equal(t, iNode, qNode)
		default:
			assert.Fail(t, "no iDev was queued")
		}

		mockEventSender.AssertExpectations(t)
	})
}

func TestZigbeeEnumerateDevice_enumerateDevice(t *testing.T) {
	t.Run("enumerating a device requests a node description and endpoints", func(t *testing.T) {
		iNode, iDev := generateTestNodeAndDevice()
		iDev.capabilities = []da.Capability{EnumerateDeviceFlag}

		expectedIEEE := iNode.ieeeAddress

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

		mockNodeQuerier := mockNodeQuerier{}
		mockNodeQuerier.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(expectedNodeDescription, nil)
		mockNodeQuerier.On("QueryNodeEndpoints", mock.Anything, expectedIEEE).Return(expectedEndpoints, nil)
		mockNodeQuerier.On("QueryNodeEndpointDescription", mock.Anything, expectedIEEE, zigbee.Endpoint(0x01)).Return(expectedEndpointDescs[0], nil)
		mockNodeQuerier.On("QueryNodeEndpointDescription", mock.Anything, expectedIEEE, zigbee.Endpoint(0x02)).Return(expectedEndpointDescs[1], nil)

		mockAdderCaller := mockAdderCaller{}
		mockAdderCaller.On("Call", mock.Anything, mock.AnythingOfType("zda.internalNodeEnumeration")).Return(nil)

		expectedStart := EnumerateDeviceStart{
			Device: iDev.toDevice(),
		}

		expectedSuccess := EnumerateDeviceSuccess{
			Device: iDev.toDevice(),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedSuccess)
		mockEventSender.On("sendEvent", expectedStart)

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("getDevice", iDev.identifier).Return(iDev, true)

		zed := ZigbeeEnumerateDevice{
			gateway:           nil,
			deviceStore:       &mockDeviceStore,
			eventSender:       &mockEventSender,
			nodeQuerier:       &mockNodeQuerier,
			internalCallbacks: &mockAdderCaller,
			queue:             nil,
			queueStop:         nil,
		}

		zed.Start()
		err := zed.Enumerate(context.TODO(), iDev.toDevice())
		assert.NoError(t, err)

		time.Sleep(20 * time.Millisecond)
		zed.Stop()

		assert.Equal(t, expectedNodeDescription, iNode.nodeDesc)
		assert.Equal(t, expectedEndpoints, iNode.endpoints)
		assert.Equal(t, expectedEndpointDescs[0], iNode.endpointDescriptions[0x01])
		assert.Equal(t, expectedEndpointDescs[1], iNode.endpointDescriptions[0x02])

		assert.False(t, iNode.supportsAPSAck)

		mockNodeQuerier.AssertExpectations(t)
		mockAdderCaller.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockDeviceStore.AssertExpectations(t)
	})

	t.Run("enumerating a device handles a failure during QueryNodeDescription", func(t *testing.T) {
		iNode, iDev := generateTestNodeAndDevice()
		iDev.capabilities = []da.Capability{EnumerateDeviceFlag}

		expectedIEEE := iNode.ieeeAddress

		expectedNodeDescription := zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		}

		expectedError := errors.New("error")

		mockNodeQuerier := mockNodeQuerier{}
		mockNodeQuerier.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(expectedNodeDescription, expectedError)

		mockAdderCaller := mockAdderCaller{}

		expectedStart := EnumerateDeviceStart{
			Device: iDev.toDevice(),
		}

		expectedFailure := EnumerateDeviceFailure{
			Device: iDev.toDevice(),
			Error:  expectedError,
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedFailure)
		mockEventSender.On("sendEvent", expectedStart)

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("getDevice", iDev.identifier).Return(iDev, true)

		zed := ZigbeeEnumerateDevice{
			gateway:           nil,
			deviceStore:       &mockDeviceStore,
			eventSender:       &mockEventSender,
			nodeQuerier:       &mockNodeQuerier,
			internalCallbacks: &mockAdderCaller,
			queue:             nil,
			queueStop:         nil,
		}

		zed.Start()
		err := zed.Enumerate(context.TODO(), iDev.toDevice())
		assert.NoError(t, err)

		time.Sleep(20 * time.Millisecond)
		zed.Stop()

		mockNodeQuerier.AssertExpectations(t)
		mockAdderCaller.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockDeviceStore.AssertExpectations(t)
	})
}

func TestZigbeeEnumerateDevice_allocateEndpointsToDevices(t *testing.T) {
	t.Run("allocating endpoints to devices results in endpoints with same device ID being mapped to the same internalDevice", func(t *testing.T) {
		iNode, iDevZero := generateTestNodeAndDevice()

		subIdZero := iDevZero.identifier.(IEEEAddressWithSubIdentifier)

		subIdOne := subIdZero
		subIdOne.SubIdentifier = 1

		iDevZero.endpoints = []zigbee.Endpoint{}
		iNode.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{}

		iDevOne := &internalDevice{
			node:               iNode,
			mutex:              &sync.RWMutex{},
			deviceID:           0,
			deviceVersion:      0,
			endpoints:          []zigbee.Endpoint{},
			productInformation: ProductInformation{},
			onOffState:         ZigbeeOnOffState{},
		}

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

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("addDevice", subIdOne, iNode).Return(iDevOne)

		zed := ZigbeeEnumerateDevice{
			deviceStore: &mockDeviceStore,
		}

		zed.allocateEndpointsToDevices(iNode)

		iNode.devices[subIdOne] = iDevOne

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, iNode.devices[subIdZero].endpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)

		mockDeviceStore.AssertExpectations(t)
	})

	t.Run("executing allocating endpoints twice does not result in duplicate endpoints", func(t *testing.T) {
		iNode, iDevZero := generateTestNodeAndDevice()

		subIdZero := iDevZero.identifier.(IEEEAddressWithSubIdentifier)

		subIdOne := subIdZero
		subIdOne.SubIdentifier = 1

		iDevZero.endpoints = []zigbee.Endpoint{}
		iNode.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{}

		iDevOne := &internalDevice{
			node:               iNode,
			mutex:              &sync.RWMutex{},
			deviceID:           0,
			deviceVersion:      0,
			endpoints:          []zigbee.Endpoint{},
			productInformation: ProductInformation{},
			onOffState:         ZigbeeOnOffState{},
		}

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

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("addDevice", subIdOne, iNode).Return(iDevOne)

		zed := ZigbeeEnumerateDevice{
			deviceStore: &mockDeviceStore,
		}

		zed.allocateEndpointsToDevices(iNode)
		zed.allocateEndpointsToDevices(iNode)

		iNode.devices[subIdOne] = iDevOne

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, iNode.devices[subIdZero].endpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)

		mockDeviceStore.AssertExpectations(t)
	})
}

func TestZigbeeEnumerateDevice_removeMissingEndpointDescriptions(t *testing.T) {
	t.Run("removes endpoint descriptions of the store if the endpoints are not in the endpoints list", func(t *testing.T) {
		zed := ZigbeeEnumerateDevice{}

		iNode, _ := generateTestNodeAndDevice()

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
		iNode, iDevs := generateTestNodeAndDevices(2)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x02}

		iDevs[0].deviceID = 0x01
		iDevs[0].endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevs[1].deviceID = 0x02
		iDevs[1].endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed := ZigbeeEnumerateDevice{}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01}, iDevs[0].endpoints)
		assert.Equal(t, []zigbee.Endpoint{0x02}, iDevs[1].endpoints)
	})

	t.Run("removes device which has had all endpoints removed", func(t *testing.T) {
		iNode, iDevs := generateTestNodeAndDevices(2)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevs[0].deviceID = 0x01
		iDevs[0].endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevs[1].deviceID = 0x02
		iDevs[1].endpoints = []zigbee.Endpoint{0x01, 0x02}

		mockDeviceStore := mockDeviceStore{}
		mockDeviceStore.On("removeDevice", iDevs[1].identifier)

		zed := ZigbeeEnumerateDevice{
			deviceStore: &mockDeviceStore,
		}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01, 0x02}, iDevs[0].endpoints)

		mockDeviceStore.AssertExpectations(t)
	})

	t.Run("does not remove sole remaining device from a node", func(t *testing.T) {
		mockDeviceStore := mockDeviceStore{}

		zed := ZigbeeEnumerateDevice{
			deviceStore: &mockDeviceStore,
		}

		iNode, iDevOne := generateTestNodeAndDevice()

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevOne.deviceID = 0x02
		iDevOne.endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{}, iDevOne.endpoints)

		mockDeviceStore.AssertExpectations(t)
	})
}
