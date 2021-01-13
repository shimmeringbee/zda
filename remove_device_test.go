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

		err := zed.Remove(context.Background(), nonSelfDevice)
		assert.Error(t, err)
	})

	t.Run("returns error if device to be enumerated does not support it", func(t *testing.T) {
		zed := ZigbeeDeviceRemoval{
			gateway: &mockGateway{},
		}

		nonCapability := da.BaseDevice{DeviceGateway: zed.gateway}

		err := zed.Remove(context.Background(), nonCapability)
		assert.Error(t, err)
	})

	t.Run("successfully calls provider with devices ieee", func(t *testing.T) {
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

		mockProvider.On("RemoveNode", mock.Anything, iDev.node.ieeeAddress).Return(nil)

		err := zed.Remove(context.Background(), device)
		assert.NoError(t, err)
	})
}
