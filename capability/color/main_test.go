package color

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*da.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.ColorFlag, impl.Capability())
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

		mockAMC := &mocks.MockAttributeMonitorCreator{}
		defer mockAMC.AssertExpectations(t)

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockZCL.On("RegisterCommandLibrary", mock.Anything)

		supervisor := zda.SimpleSupervisor{
			AttributeMonitorCreatorImpl: mockAMC,
			ZCLImpl:                     mockZCL,
		}

		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.ColorMode, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.CurrentX, zcl.TypeUnsignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.CurrentY, zcl.TypeUnsignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.CurrentHue, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.CurrentSaturation, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.ColorControlId, color_control.ColorTemperatureMireds, zcl.TypeUnsignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})

		impl.Init(supervisor)
	})
}
