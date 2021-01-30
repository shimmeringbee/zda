package color

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
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

func TestImplementation_attributeUpdate(t *testing.T) {
	t.Run("updates bulb color mode state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        0,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.ColorMode,
				Color: capabilities.ColorSettings{
					Current: color.XYColor{
						X:  0,
						Y:  0,
						Y2: 100,
					},
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.ColorMode, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeEnum8,
			Value:    uint8(1),
		})

		assert.Equal(t, uint8(1), i.data[addr].State.CurrentMode)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})

	t.Run("updates bulb CurrentX state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        1,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.ColorMode,
				Color: capabilities.ColorSettings{
					Current: color.XYColor{
						X:  0.00152587890625,
						Y:  0,
						Y2: 100,
					},
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.CurrentX, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(100),
		})

		assert.InDelta(t, 0.0015, i.data[addr].State.CurrentX, 0.0001)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})

	t.Run("updates bulb CurrentY state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        1,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.ColorMode,
				Color: capabilities.ColorSettings{
					Current: color.XYColor{
						X:  0,
						Y:  0.0030517578125,
						Y2: 100,
					},
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.CurrentY, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(200),
		})

		assert.InDelta(t, 0.0030, i.data[addr].State.CurrentY, 0.0001)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})

	t.Run("updates bulb CurrentHue state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        0,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.ColorMode,
				Color: capabilities.ColorSettings{
					Current: color.HSVColor{
						Hue:   283.46456692913387,
						Sat:   0,
						Value: 1,
					},
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.CurrentHue, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint64(200),
		})

		assert.InDelta(t, 283.4, i.data[addr].State.CurrentHue, 0.1)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})

	t.Run("updates bulb CurrentSaturation state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        0,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.ColorMode,
				Color: capabilities.ColorSettings{
					Current: color.HSVColor{
						Hue:   0,
						Sat:   0.7874015748031497,
						Value: 1,
					},
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.CurrentSaturation, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint64(200),
		})

		assert.InDelta(t, 0.787, i.data[addr].State.CurrentSat, 0.001)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})

	t.Run("updates bulb CurrentTemperature state and sends an event when attribute is updated by monitor", func(t *testing.T) {
		addr := zda.IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		i := &Implementation{}
		i.data = map[zda.IEEEAddressWithSubIdentifier]Data{
			addr: {
				State: State{
					CurrentMode:        2,
					CurrentX:           0,
					CurrentY:           0,
					CurrentHue:         0,
					CurrentSat:         0,
					CurrentTemperature: 0,
				},
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
					InClusterList: []zigbee.ClusterID{zcl.ColorControlId},
				},
			},
		}

		mockDAES.On("Send", capabilities.ColorStatusUpdate{
			Device: i.supervisor.ComposeDADevice().Compose(device),
			State: capabilities.ColorStatus{
				Mode: capabilities.TemperatureMode,
				Temperature: capabilities.TemperatureSettings{
					Current: 5000.0,
				},
			},
		})

		currentTime := time.Now()

		i.attributeUpdate(device, color_control.ColorTemperatureMireds, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(200),
		})

		assert.InDelta(t, 5000.0, i.data[addr].State.CurrentTemperature, 0.1)
		assert.True(t, i.data[addr].LastChangeTime.Equal(currentTime) || i.data[addr].LastChangeTime.After(currentTime))
		assert.True(t, i.data[addr].LastUpdateTime.Equal(currentTime) || i.data[addr].LastUpdateTime.After(currentTime))
	})
}
