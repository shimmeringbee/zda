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

func TestZigbeeLocalDebugCapabilities_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.EnumerateDevice", func(t *testing.T) {
		assert.Implements(t, (*LocalDebug)(nil), new(ZigbeeLocalDebug))
	})
}

func TestZigbeeLocalDebugCapabilities_ReturnsLocalDebugCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(LocalDebugFlag)
		assert.IsType(t, (*ZigbeeLocalDebug)(nil), actualZdd)
	})
}

func TestZigbeeLocalDebugCapabilities_Start(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zld := zgw.capabilities[LocalDebugFlag].(*ZigbeeLocalDebug)
		nonSelfDevice := da.BaseDevice{}

		err := zld.Start(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zld := zgw.capabilities[LocalDebugFlag].(*ZigbeeLocalDebug)
		nonCapability := da.BaseDevice{DeviceGateway: zgw}

		err := zld.Start(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("sends an event with the debug in after a successful request", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		zld := zgw.capabilities[LocalDebugFlag].(*ZigbeeLocalDebug)

		expectedIEEEAddress := zigbee.IEEEAddress(0x0102030405060708)
		node := zgw.addNode(expectedIEEEAddress)
		expectedDevId := IEEEAddressWithSubIdentifier{
			IEEEAddress:   expectedIEEEAddress,
			SubIdentifier: 0x7f,
		}

		node.endpoints = []zigbee.Endpoint{0x01, 0x02}

		device := zgw.addDevice(expectedDevId, node)
		device.endpoints = []zigbee.Endpoint{0x01}
		device.deviceID = 0x02
		device.deviceVersion = 0x03

		expectedDebug := LocalDebugNodeData{
			IEEEAddress:          expectedIEEEAddress.String(),
			NodeDescription:      zigbee.NodeDescription{},
			Endpoints:            []int{0x01, 0x02},
			EndpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
			Devices: map[string]LocalDebugDeviceData{expectedDevId.String(): {
				Identifier:        expectedDevId.String(),
				AssignedEndpoints: []int{0x01},
				DeviceId:          0x02,
				DeviceVersion:     0x03,
			}},
		}

		err := zld.Start(context.Background(), device.toDevice())
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		_, err = zgw.ReadEvent(ctx)
		assert.NoError(t, err)

		event, _ := zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		start := event.(LocalDebugStart)
		assert.Equal(t, device.toDevice(), start.Device)

		event, _ = zgw.ReadEvent(ctx)
		assert.NotNil(t, event)

		success := event.(LocalDebugSuccess)
		assert.Equal(t, device.toDevice(), success.Device)
		assert.Equal(t, expectedDebug, success.Debug)
		assert.Equal(t, ZigbeeLocalDebugMediaType, success.MediaType)
	})
}
