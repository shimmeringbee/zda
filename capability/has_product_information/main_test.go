package has_product_information

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*zda.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.HasProductInformationFlag, impl.Capability())
	})
}

func TestImplementation_InitableCapability(t *testing.T) {
	t.Run("matches the InitableCapability interface", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*zda.InitableCapability)(nil), impl)
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("subscribes to events", func(t *testing.T) {
		impl := &Implementation{}

		mockEventSubscription := &mocks.MockEventSubscription{}

		supervisor := zda.SimpleSupervisor{
			ESImpl: mockEventSubscription,
		}

		mockEventSubscription.On("AddedDeviceEvent", mock.Anything)
		mockEventSubscription.On("RemovedDeviceEvent", mock.Anything)
		mockEventSubscription.On("EnumerateDeviceEvent", mock.Anything)
		defer mockEventSubscription.AssertExpectations(t)

		impl.Init(supervisor)
	})
}
