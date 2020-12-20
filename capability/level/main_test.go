package level

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/level"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*da.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.LevelFlag, impl.Capability())
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

		mockAMC.On("Create", impl, zcl.LevelControlId, level.CurrentLevel, zcl.TypeUnsignedInt8, mock.Anything).Return(&mocks.MockAttributeMonitor{})

		impl.Init(supervisor)
	})
}

func TestImplementation_attributeUpdate(t *testing.T) {
	t.Run("updates state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State:           0.0,
				RequiresPolling: false,
				Endpoint:        0,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockDAES := mocks.MockDAEventSender{}
		defer mockDAES.AssertExpectations(t)

		i.supervisor = &zda.SimpleSupervisor{
			CDADImpl:   &zda.ComposeDADeviceShim{},
			DAESImpl:   &mockDAES,
			LoggerImpl: logwrap.New(discard.Discard()),
		}

		endpoint := zigbee.Endpoint(0x01)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.LevelControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.LevelStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.LevelStatus{
				CurrentLevel: 1.0,
			},
		})

		i.attributeUpdate(device, level.CurrentLevel, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint8(0xfe),
		})

		assert.Equal(t, 1.0, i.data[addr].State)
	})
}
