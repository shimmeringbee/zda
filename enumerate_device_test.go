package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		device := da.Device{Gateway: zgw, Capabilities: []da.Capability{EnumerateDeviceFlag}}

		// Stop the worker routines so that we can examine the queue, with 50ms cooldown to allow end.
		zed.Stop()
		time.Sleep(50 * time.Millisecond)

		err := zed.Enumerate(context.Background(), device)
		assert.NoError(t, err)

		select {
		case qDevice := <-zed.queue:
			assert.Equal(t, device, qDevice)
		default:
			assert.Fail(t, "no device was queued")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		startEvent := event.(EnumerateDeviceStart)
		assert.Equal(t, device, startEvent.Device)
	})

	t.Run("queues a device for enumeration on internal join message", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		zgw.Start()
		defer stop(t)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		device := da.Device{Gateway: zgw, Capabilities: []da.Capability{EnumerateDeviceFlag}}

		// Stop the worker routines so that we can examine the queue, with 50ms cooldown to allow end.
		zed.Stop()
		time.Sleep(50 * time.Millisecond)

		err := zgw.callbacks.Call(context.Background(), internalNodeJoin{node: &ZigbeeDevice{device: device}})
		assert.NoError(t, err)

		select {
		case qDevice := <-zed.queue:
			assert.Equal(t, device, qDevice)
		default:
			assert.Fail(t, "no device was queued")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		startEvent := event.(EnumerateDeviceStart)
		assert.Equal(t, device, startEvent.Device)
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
				DeviceID:       0x04,
				DeviceVersion:  2,
				InClusterList:  nil,
				OutClusterList: nil,
			},
		}

		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
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

		zDev := zgw.addDevice(expectedIEEE)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		err := zed.Enumerate(context.Background(), zDev.device)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, _ = zgw.ReadEvent(ctx)
		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		successEvent := event.(EnumerateDeviceSuccess)
		assert.Equal(t, expectedIEEE, successEvent.Device.Identifier)

		assert.Equal(t, expectedNodeDescription, zDev.nodeDesc)
		assert.Equal(t, expectedEndpoints, zDev.endpoints)
		assert.Equal(t, expectedEndpointDescs[0], zDev.endpointDescriptions[0x01])
		assert.Equal(t, expectedEndpointDescs[1], zDev.endpointDescriptions[0x02])

		assert.True(t, callbackCalled)
	})

	t.Run("enumerating a device handles a failure during QueryNodeDescription", func(t *testing.T) {
		expectedError := errors.New("expected error")
		expectedIEEE := zigbee.IEEEAddress(0x00112233445566)

		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("QueryNodeDescription", mock.Anything, expectedIEEE).Return(zigbee.NodeDescription{}, expectedError)
		zgw.Start()
		defer stop(t)

		zDev := zgw.addDevice(expectedIEEE)

		zed := zgw.capabilities[EnumerateDeviceFlag].(*ZigbeeEnumerateDevice)
		err := zed.Enumerate(context.Background(), zDev.device)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, _ = zgw.ReadEvent(ctx)
		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		failureEvent := event.(EnumerateDeviceFailure)
		assert.Equal(t, expectedIEEE, failureEvent.Device.Identifier)
		assert.Equal(t, expectedError, failureEvent.Error)
	})
}
