package relative_humidity_sensor

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/relative_humidity_measurement"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestImplementation_Capability(t *testing.T) {
	t.Run("matches the CapabiltiyBasic interface and returns the correct Capability", func(t *testing.T) {
		impl := &Implementation{}

		assert.Implements(t, (*da.BasicCapability)(nil), impl)
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

		mockAMC := &mocks.MockAttributeMonitorCreator{}
		defer mockAMC.AssertExpectations(t)

		supervisor := zda.SimpleSupervisor{
			AttributeMonitorCreatorImpl: mockAMC,
		}

		mockAMC.On("Create", impl, zcl.RelativeHumidityMeasurementId, relative_humidity_measurement.MeasuredValue, zcl.TypeUnsignedInt16, mock.Anything).Return(&mocks.MockAttributeMonitor{})
		mockAMC.On("Create", impl, zcl.BasicId, zcl.AttributeID(0xff01), zcl.TypeStringCharacter8, mock.Anything).Return(&mocks.MockAttributeMonitor{})

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
					InClusterList: []zigbee.ClusterID{zcl.RelativeHumidityMeasurementId},
				},
			},
		}

		mockDAES.On("Send", capabilities.RelativeHumiditySensorState{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State:  []capabilities.RelativeHumidityReading{{Value: 0.0001}},
		})

		currentTime := time.Now()

		i.attributeUpdateRelativeHumidityMeasurementCluster(device, relative_humidity_measurement.MeasuredValue, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(1),
		})

		assert.Equal(t, 0.0001, i.data[addr].State)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})
}

func TestImplementation_attributeUpdateVendorXiaomiApproachOne(t *testing.T) {
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
			CDADImpl:   &zda.ComposeDADeviceShim{},
			DAESImpl:   &mockDAES,
			LoggerImpl: logwrap.New(discard.Discard()),
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

		mockDAES.On("Send", capabilities.RelativeHumiditySensorState{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State:  []capabilities.RelativeHumidityReading{{Value: 0.4452}},
		})

		currentTime := time.Now()

		i.attributeUpdateVendorXiaomiApproachOne(device, zcl.AttributeID(0xff01), zcl.AttributeDataTypeValue{
			DataType: zcl.TypeStringCharacter8,
			Value:    string([]byte{0x65, 0x21, 0x64, 0x11}),
		})

		assert.Equal(t, 0.4452, i.data[addr].State)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})
}
