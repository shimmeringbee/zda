package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestZigbeeEnumerateCapabilities_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.EnumerateDeviceEvent", func(t *testing.T) {
		assert.Implements(t, (*capabilities.EnumerateDevice)(nil), new(ZigbeeEnumerateDevice))
	})
}

func TestZigbeeEnumerateCapabilities_Init(t *testing.T) {
	t.Run("registers against callbacks", func(t *testing.T) {
		mockAdderCaller := MockAdderCaller{}

		mockAdderCaller.On("Add", mock.AnythingOfType("func(context.Context, zda.internalNodeJoin) error"))

		zed := ZigbeeEnumerateDevice{
			internalCallbacks: &mockAdderCaller,
		}

		zed.Init(nil)

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

		nt, iNode, iDev := generateNodeTableWithData(1)

		iDev[0].capabilities = []da.Capability{capabilities.EnumerateDeviceFlag}

		expectedEvent := capabilities.EnumerateDeviceStart{
			Device: iDev[0].toDevice(&mockGateway),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedEvent)

		zed := ZigbeeEnumerateDevice{
			gateway:     &mockGateway,
			nodeTable:   nt,
			eventSender: &mockEventSender,
			logger:      lw.New(discard.Discard()),
		}

		// Stop and start zed to populate queues
		zed.Start()
		zed.Stop()
		time.Sleep(time.Millisecond)

		err := zed.Enumerate(context.Background(), iDev[0].toDevice(&mockGateway))
		assert.NoError(t, err)

		status, err := zed.Status(context.Background(), iDev[0].toDevice(&mockGateway))
		assert.NoError(t, err)
		assert.True(t, status.Enumerating)

		select {
		case qNode := <-zed.queue:
			assert.Equal(t, iNode, qNode)
		default:
			assert.Fail(t, "no iDev was queued")
		}

		mockEventSender.AssertExpectations(t)
	})

	t.Run("queues a device for enumeration on internal join message", func(t *testing.T) {
		mockGateway := mockGateway{}

		_, iNode, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]

		expectedEvent := capabilities.EnumerateDeviceStart{
			Device: iDev.toDevice(&mockGateway),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedEvent)

		zed := ZigbeeEnumerateDevice{
			gateway:     &mockGateway,
			eventSender: &mockEventSender,
			logger:      lw.New(discard.Discard()),
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
		nt, iNode, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]
		iDev.capabilities = []da.Capability{capabilities.EnumerateDeviceFlag}

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

		mockAdderCaller := MockAdderCaller{}
		mockAdderCaller.On("Call", mock.Anything, mock.AnythingOfType("zda.internalNodeEnumeration")).Return(nil)
		mockAdderCaller.On("Call", mock.Anything, mock.AnythingOfType("zda.internalDeviceEnumeration")).Return(nil)

		expectedStart := capabilities.EnumerateDeviceStart{
			Device: iDev.toDevice(nil),
		}

		expectedSuccess := capabilities.EnumerateDeviceSuccess{
			Device: iDev.toDevice(nil),
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedSuccess)
		mockEventSender.On("sendEvent", expectedStart)

		zed := ZigbeeEnumerateDevice{
			gateway:           nil,
			nodeTable:         nt,
			eventSender:       &mockEventSender,
			nodeQuerier:       &mockNodeQuerier,
			internalCallbacks: &mockAdderCaller,
			queue:             nil,
			queueStop:         nil,
			logger:            lw.New(discard.Discard()),
		}

		zed.Start()
		err := zed.Enumerate(context.TODO(), iDev.toDevice(nil))
		assert.NoError(t, err)

		time.Sleep(20 * time.Millisecond)
		zed.Stop()

		assert.Equal(t, expectedNodeDescription, iNode.nodeDesc)
		assert.Equal(t, expectedEndpoints, iNode.endpoints)
		assert.Equal(t, expectedEndpointDescs[0], iNode.endpointDescriptions[0x01])
		assert.Equal(t, expectedEndpointDescs[1], iNode.endpointDescriptions[0x02])

		assert.False(t, iNode.supportsAPSAck)

		status, err := zed.Status(context.Background(), iDev.toDevice(nil))
		assert.NoError(t, err)
		assert.False(t, status.Enumerating)

		mockNodeQuerier.AssertExpectations(t)
		mockAdderCaller.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
	})

	t.Run("enumerating a device handles a failure during QueryNodeDescription", func(t *testing.T) {
		nt, iNode, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]

		iDev.capabilities = []da.Capability{capabilities.EnumerateDeviceFlag}

		expectedIEEE := iNode.ieeeAddress

		expectedNodeDescription := zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		}

		expectedError := errors.New("error")

		mockNodeQuerier := mockNodeQuerier{}
		mockNodeQuerier.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(expectedNodeDescription, expectedError)

		mockAdderCaller := MockAdderCaller{}

		expectedStart := capabilities.EnumerateDeviceStart{
			Device: iDev.toDevice(nil),
		}

		expectedFailure := capabilities.EnumerateDeviceFailure{
			Device: iDev.toDevice(nil),
			Error:  expectedError,
		}

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", expectedFailure)
		mockEventSender.On("sendEvent", expectedStart)

		zed := ZigbeeEnumerateDevice{
			gateway:           nil,
			nodeTable:         nt,
			eventSender:       &mockEventSender,
			nodeQuerier:       &mockNodeQuerier,
			internalCallbacks: &mockAdderCaller,
			queue:             nil,
			queueStop:         nil,
			logger:            lw.New(discard.Discard()),
		}

		zed.Start()
		err := zed.Enumerate(context.TODO(), iDev.toDevice(nil))
		assert.NoError(t, err)

		time.Sleep(20 * time.Millisecond)
		zed.Stop()

		mockNodeQuerier.AssertExpectations(t)
		mockAdderCaller.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
	})
}

func TestZigbeeEnumerateDevice_allocateEndpointsToDevices(t *testing.T) {
	t.Run("allocating endpoints to devices results in endpoints with same device ID being mapped to the same internalDevice", func(t *testing.T) {
		nt, iNode, iDevs := generateNodeTableWithData(2)
		iDevZero := iDevs[0]
		iDevOne := iDevs[1]

		iDevs[0].endpoints = []zigbee.Endpoint{}
		iDevs[1].endpoints = []zigbee.Endpoint{}
		iDevs[0].deviceID = 0x10
		iDevs[1].deviceID = 0
		iNode.endpointDescriptions = make(map[zigbee.Endpoint]zigbee.EndpointDescription)

		subIdZero := iDevZero.subidentifier
		subIdOne := iDevOne.subidentifier

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

		zed := ZigbeeEnumerateDevice{
			nodeTable: nt,
		}

		zed.allocateEndpointsToDevices(iNode)

		// Test code required due to the non deterministic nature of maps.
		if iNode.devices[subIdZero].endpoints[0] == 0x20 {
			t := subIdZero
			subIdZero = subIdOne
			subIdOne = t
		}

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, iNode.devices[subIdZero].endpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)
	})

	t.Run("executing allocating endpoints twice does not result in duplicate endpoints", func(t *testing.T) {
		nt, iNode, iDevs := generateNodeTableWithData(2)
		iDevZero := iDevs[0]
		iDevOne := iDevs[1]

		iDevs[0].endpoints = []zigbee.Endpoint{}
		iDevs[1].endpoints = []zigbee.Endpoint{}
		iDevs[0].deviceID = 0
		iDevs[1].deviceID = 0
		iNode.endpointDescriptions = make(map[zigbee.Endpoint]zigbee.EndpointDescription)

		subIdZero := iDevZero.subidentifier
		subIdOne := iDevOne.subidentifier

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

		zed := ZigbeeEnumerateDevice{
			nodeTable: nt,
		}

		zed.allocateEndpointsToDevices(iNode)
		zed.allocateEndpointsToDevices(iNode)

		iNode.devices[subIdOne] = iDevOne

		// Test code required due to the non deterministic nature of maps.
		if iNode.devices[subIdZero].endpoints[0] == 0x20 {
			t := subIdZero
			subIdZero = subIdOne
			subIdOne = t
		}

		assert.Equal(t, []zigbee.Endpoint{0x10, 0x11}, iNode.devices[subIdZero].endpoints)
		assert.Equal(t, uint16(0x10), iNode.devices[subIdZero].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdZero].deviceVersion)

		assert.Equal(t, []zigbee.Endpoint{0x20}, iNode.devices[subIdOne].endpoints)
		assert.Equal(t, uint16(0x20), iNode.devices[subIdOne].deviceID)
		assert.Equal(t, uint8(1), iNode.devices[subIdOne].deviceVersion)
	})
}

func TestZigbeeEnumerateDevice_removeMissingEndpointDescriptions(t *testing.T) {
	t.Run("removes endpoint descriptions of the store if the endpoints are not in the endpoints list", func(t *testing.T) {
		zed := ZigbeeEnumerateDevice{}

		_, iNode, _ := generateNodeTableWithData(1)

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
		nt, iNode, iDevs := generateNodeTableWithData(2)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x02}

		iDevs[0].deviceID = 0x01
		iDevs[0].endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevs[1].deviceID = 0x02
		iDevs[1].endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed := ZigbeeEnumerateDevice{
			nodeTable: nt,
		}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01}, iDevs[0].endpoints)
		assert.Equal(t, []zigbee.Endpoint{0x02}, iDevs[1].endpoints)
	})

	t.Run("removes device which has had all endpoints removed", func(t *testing.T) {
		nt, iNode, iDevs := generateNodeTableWithData(2)

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevs[0].deviceID = 0x01
		iDevs[0].endpoints = []zigbee.Endpoint{0x01, 0x02}

		iDevs[1].deviceID = 0x02
		iDevs[1].endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed := ZigbeeEnumerateDevice{
			nodeTable: nt,
		}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{0x01, 0x02}, iDevs[0].endpoints)

		assert.Equal(t, len(iNode.devices), 1)
	})

	t.Run("does not remove sole remaining device from a node", func(t *testing.T) {
		nt, iNode, iDevs := generateNodeTableWithData(1)
		iDevZero := iDevs[0]

		zed := ZigbeeEnumerateDevice{
			nodeTable: nt,
		}

		iNode.endpoints = []zigbee.Endpoint{0x01}
		iNode.endpointDescriptions[0x01] = zigbee.EndpointDescription{DeviceID: 0x01}
		iNode.endpointDescriptions[0x02] = zigbee.EndpointDescription{DeviceID: 0x01}

		iDevZero.deviceID = 0x02
		iDevZero.endpoints = []zigbee.Endpoint{0x01, 0x02}

		zed.deallocateDevicesFromMissingEndpoints(iNode)

		assert.Equal(t, []zigbee.Endpoint{}, iDevZero.endpoints)

		assert.Equal(t, len(iNode.devices), 1)
	})
}
