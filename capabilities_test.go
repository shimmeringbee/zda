package zda

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestZigbeeGateway_ReturnsDeviceDiscoveryCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(capabilities.DeviceDiscoveryFlag)
		assert.IsType(t, (*ZigbeeDeviceDiscovery)(nil), actualZdd)
	})
}

func TestZigbeeGateway_ReturnsHasProductInformationCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(capabilities.HasProductInformationFlag)
		assert.IsType(t, (*ZigbeeHasProductInformation)(nil), actualZdd)
	})
}

func TestZigbeeGateway_ReturnsOnOffCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZOO := zgw.Capability(capabilities.OnOffFlag)
		assert.IsType(t, (*ZigbeeOnOff)(nil), actualZOO)
	})
}

func TestZigbeeGateway_ReturnsEnumerateCapabilitiesCapability(t *testing.T) {
	t.Run("returns capability on query", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		actualZdd := zgw.Capability(capabilities.EnumerateDeviceFlag)
		assert.IsType(t, (*ZigbeeEnumerateDevice)(nil), actualZdd)
	})
}
