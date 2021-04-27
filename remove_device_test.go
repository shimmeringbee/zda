package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestZigbeeDeviceRemoval_Remove(t *testing.T) {
	t.Run("returns error if device to be enumerated does not belong to gateway", func(t *testing.T) {
		zed := ZigbeeDeviceRemoval{
			gateway: &mockGateway{},
		}

		nonSelfDevice := da.BaseDevice{}

		err := zed.Remove(context.Background(), nonSelfDevice, capabilities.Request)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zed := ZigbeeDeviceRemoval{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zed.gateway}

		err := zed.Remove(context.Background(), nonCapability, capabilities.Request)
		assert.Error(t, err)
	})

	t.Run("successfully calls RequestNodeLeave on provider with devices ieee and Request flag", func(t *testing.T) {
		nt, _, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]
		iDev.capabilities = []da.Capability{capabilities.DeviceRemovalFlag}

		mockProvider := zigbee.MockProvider{}
		defer mockProvider.AssertExpectations(t)

		zed := ZigbeeDeviceRemoval{
			gateway:     &mockGateway{},
			nodeRemover: &mockProvider,
			nodeTable:   nt,
			logger:      lw.New(discard.Discard()),
		}

		device := da.BaseDevice{
			DeviceGateway:      zed.gateway,
			DeviceIdentifier:   iDev.generateIdentifier(),
			DeviceCapabilities: iDev.capabilities,
		}

		mockProvider.On("RequestNodeLeave", mock.Anything, iDev.node.ieeeAddress).Return(nil)

		err := zed.Remove(context.Background(), device, capabilities.Request)
		assert.NoError(t, err)
	})

	t.Run("successfully calls ForceNodeLeave on provider with devices ieee and Force flag", func(t *testing.T) {
		nt, _, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]
		iDev.capabilities = []da.Capability{capabilities.DeviceRemovalFlag}

		mockProvider := zigbee.MockProvider{}
		defer mockProvider.AssertExpectations(t)

		zed := ZigbeeDeviceRemoval{
			gateway:     &mockGateway{},
			nodeRemover: &mockProvider,
			nodeTable:   nt,
			logger:      lw.New(discard.Discard()),
		}

		device := da.BaseDevice{
			DeviceGateway:      zed.gateway,
			DeviceIdentifier:   iDev.generateIdentifier(),
			DeviceCapabilities: iDev.capabilities,
		}

		mockProvider.On("ForceNodeLeave", mock.Anything, iDev.node.ieeeAddress).Return(nil)

		err := zed.Remove(context.Background(), device, capabilities.Force)
		assert.NoError(t, err)
	})
}
