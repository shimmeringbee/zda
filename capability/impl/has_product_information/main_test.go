package has_product_information

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zda/capability/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capability.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.HasProductInformationFlag, impl.Capability())
	})
}

func TestImplementation_InitableCapability(t *testing.T) {
	t.Run("matches the InitableCapability interface", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*capability.InitableCapability)(nil), impl)
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("subscribes to events", func(t *testing.T) {
		impl := &Implementation{}

		mockEventSubscription := &mocks.MockEventSubscription{}

		supervisor := capability.SimpleSupervisor{
			ESImpl: mockEventSubscription,
		}

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)
		defer mockEventSubscription.AssertExpectations(t)

		impl.Init(supervisor)
	})
}
