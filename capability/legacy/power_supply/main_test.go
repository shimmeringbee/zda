package power_supply

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
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
		assert.Equal(t, capabilities.PowerSupplyFlag, impl.Capability())
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

		supervisor := zda.SimpleSupervisor{
			AttributeMonitorCreatorImpl: mockAMC,
		}

		mockAMC.On("Create", impl, zcl.PowerConfigurationId, power_configuration.MainsVoltage, zcl.TypeUnsignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.PowerConfigurationId, power_configuration.MainsFrequency, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.PowerConfigurationId, power_configuration.BatteryVoltage, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.PowerConfigurationId, power_configuration.BatteryPercentageRemaining, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.BasicId, zcl.AttributeID(0xff01), zcl.TypeStringCharacter8, mock.Anything).Return(&mocks.MockAttributeMonitor{})

		impl.Init(supervisor)
	})
}
