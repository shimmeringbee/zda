package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
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
		defer stop(t)

		actualZdd := zgw.Capability(EnumerateDeviceFlag)
		assert.IsType(t, (*ZigbeeEnumerateDevice)(nil), actualZdd)
	})
}

func TestZigbeeEnumerateCapabilities_Enumerate(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		defer stop(t)

		sed := ZigbeeEnumerateDevice{gateway: zgw}
		nonSelfDevice := da.Device{}

		err := sed.Enumerate(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		defer stop(t)

		zed := ZigbeeEnumerateDevice{gateway: zgw}
		nonSelfDevice := da.Device{Gateway: zgw}

		err := zed.Enumerate(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})
}
