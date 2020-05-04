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
	mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
	zgw := New(mockProvider)

	zgw.Start()

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
		zgw, _, stop := NewTestZigbeeGateway()
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
		zgw, _, stop := NewTestZigbeeGateway()
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
		zgw, _, stop := NewTestZigbeeGateway()
		defer stop(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := zgw.ReadEvent(ctx)
		assert.Error(t, err)
	})

	t.Run("sent events are received through ReadEvent", func(t *testing.T) {
		zgw, _, stop := NewTestZigbeeGateway()
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
