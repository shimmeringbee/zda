package temperature_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
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
		assert.Equal(t, capabilities.TemperatureSensorFlag, impl.Capability())
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

		mockAMC.On("Create", impl, zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})

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
				State:           0,
				RequiresPolling: false,
				Endpoint:        0,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockDAES := mocks.MockDAEventSender{}

		i.supervisor = &zda.SimpleSupervisor{
			CDADImpl: &zda.ComposeDADeviceShim{},
			DAESImpl: &mockDAES,
		}

		mockDAES.AssertExpectations(t)

		endpoint := zigbee.Endpoint(0x01)

		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.TemperatureMeasurementId},
				},
			},
		}

		mockDAES.On("Send", capabilities.TemperatureSensorState{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State:  []capabilities.TemperatureReading{{Value: 274.15}},
		})

		i.attributeUpdate(device, temperature_measurement.MeasuredValue, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeSignedInt16,
			Value:    int64(100),
		})

		assert.Equal(t, 274.15, i.data[addr].State)
	})
}
