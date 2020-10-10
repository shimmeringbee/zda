package relative_humidity_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/relative_humidity_measurement"
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

		assert.Implements(t, (*zda.BasicCapability)(nil), impl)
		assert.Equal(t, capabilities.RelativeHumiditySensorFlag, impl.Capability())
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

		mockZCL := &mocks.MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockZCL.On("Listen", mock.AnythingOfType("ZCLFilter"), mock.AnythingOfType("ZCLCallback"))

		supervisor := zda.SimpleSupervisor{
			ZCLImpl: mockZCL,
		}

		impl.Init(supervisor)
	})
}

func TestImplementation_pollDevice(t *testing.T) {
	t.Run("reads from device, and sets state", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}
		endpoint := zigbee.Endpoint(0x11)

		i := &Implementation{}

		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State:           0,
				RequiresPolling: false,
				Endpoint:        endpoint,
			},
		}
		i.datalock = &sync.RWMutex{}

		mockZCL := &mocks.MockZCL{}
		mockDAES := mocks.MockDAEventSender{}
		mockCDAD := mocks.MockComposeDADevice{}
		defer mockDAES.AssertExpectations(t)
		defer mockCDAD.AssertExpectations(t)
		defer mockZCL.AssertExpectations(t)

		i.supervisor = zda.SimpleSupervisor{
			ZCLImpl:  mockZCL,
			DAESImpl: &mockDAES,
			CDADImpl: &mockCDAD,
		}

		daDevice := da.BaseDevice{}
		device := zda.Device{
			Identifier:   addr,
			Capabilities: []da.Capability{},
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				endpoint: {
					Endpoint:      endpoint,
					InClusterList: []zigbee.ClusterID{zcl.RelativeHumidityMeasurementId},
				},
			},
		}

		mockCDAD.On("Compose", device).Return(daDevice)
		mockDAES.On("Send", capabilities.RelativeHumiditySensorState{
			Device: daDevice,
			State:  []capabilities.RelativeHumidityReading{{Value: 0.01}},
		})

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, zcl.RelativeHumidityMeasurementId, []zcl.AttributeID{relative_humidity_measurement.MeasuredValue}).Return(
			map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				relative_humidity_measurement.MeasuredValue: {
					Identifier: relative_humidity_measurement.MeasuredValue,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: zcl.TypeUnsignedInt16,
						Value:    uint64(100),
					},
				},
			}, nil)

		i.pollDevice(context.Background(), device)

		assert.Equal(t, 0.01, i.data[addr].State)
	})
}
