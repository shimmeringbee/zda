package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestCapabilityManager_Add(t *testing.T) {
	t.Run("makes it available in the capability by flag store", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)
		actualCapability := m.Get(f)

		assert.Equal(t, mC, actualCapability)
	})

	t.Run("adds a persisting capability to a key based store", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)
		actualCapabilities := m.PersistingCapabilities()

		assert.Equal(t, mC, actualCapabilities[kN])
	})

	t.Run("adds a device managing capability to a slice", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)
		assert.Contains(t, m.deviceManagerCapability, mC)
	})

	t.Run("adds a device enumerating capability to a slice", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)
		assert.Contains(t, m.deviceEnumerationCapability, mC)
	})
}

func TestCapabilityManager_Init(t *testing.T) {
	t.Run("inits any added capabilities that support it", func(t *testing.T) {
		mAC := &MockAdderCaller{}
		mAC.On("Add", mock.Anything)

		m := CapabilityManager{
			callbackAdder:       mAC,
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)
		mC.On("Init", mock.Anything)

		m.Add(mC)
		m.Init()

		assert.NotNil(t, mC.Calls[2].Arguments.Get(0))
	})

	t.Run("registers device callbacks on init", func(t *testing.T) {
		mAC := &MockAdderCaller{}
		mAC.On("Add", mock.Anything).Times(3)
		defer mAC.AssertExpectations(t)

		m := CapabilityManager{
			callbackAdder:       mAC,
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		m.Init()
	})
}

func TestCapabilityManager_StartStop(t *testing.T) {
	t.Run("starts any added capabilities that support it", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)
		mC.On("Start")

		m.Add(mC)
		m.Start()
	})

	t.Run("stops any added capabilities that support it", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)
		mC.On("Stop")

		m.Add(mC)
		m.Stop()
	})
}

type mockCapability struct {
	mock.Mock
}

func (m *mockCapability) Capability() da.Capability {
	args := m.Called()
	return args.Get(0).(da.Capability)
}

func (m *mockCapability) Init(s CapabilitySupervisor) {
	m.Called(s)
}

func (m *mockCapability) Start() {
	m.Called()
}

func (m *mockCapability) Stop() {
	m.Called()
}

func (m *mockCapability) KeyName() string {
	args := m.Called()
	return args.String(0)
}
func (m *mockCapability) DataStruct() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *mockCapability) Save(d Device) (interface{}, error) {
	args := m.Called(d)
	return args.Get(0), args.Error(1)
}

func (m *mockCapability) Load(d Device, s interface{}) error {
	args := m.Called(d, s)
	return args.Error(0)
}

func (m *mockCapability) AddedDevice(c context.Context, d Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func (m *mockCapability) RemovedDevice(c context.Context, d Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func (m *mockCapability) EnumerateDevice(c context.Context, d Device) error {
	args := m.Called(c, d)
	return args.Error(0)
}

func TestCapabilityManager_initSupervisor_FetchCapability(t *testing.T) {
	t.Run("provides the parent capability manager as the implementation", func(t *testing.T) {
		m := CapabilityManager{}

		s := m.initSupervisor()

		assert.Equal(t, &m, s.FetchCapability())
	})
}

func TestCapabilityManager_initSupervisor_ManageDeviceCapabilities(t *testing.T) {
	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mdcm := &mockDeviceCapabilityManager{}
		defer mdcm.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{
			Identifier: addr,
		}
		c := da.Capability(0x0001)

		mdcm.On("AddCapability", addr, c)

		m := CapabilityManager{deviceCapabilityManager: mdcm}
		s := m.initSupervisor()

		s.ManageDeviceCapabilities().Add(zdaDevice, c)
	})

	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mdcm := &mockDeviceCapabilityManager{}
		defer mdcm.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{
			Identifier: addr,
		}
		c := da.Capability(0x0001)

		mdcm.On("RemoveCapability", addr, c)

		m := CapabilityManager{deviceCapabilityManager: mdcm}
		s := m.initSupervisor()

		s.ManageDeviceCapabilities().Remove(zdaDevice, c)
	})
}

func TestCapabilityManager_initSupervisor_DAEventSender(t *testing.T) {
	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mes := &mockEventSender{}
		defer mes.AssertExpectations(t)

		mes.On("sendEvent", mock.Anything)

		m := CapabilityManager{eventSender: mes}
		s := m.initSupervisor()

		s.DAEventSender().Send(nil)
	})
}

func TestCapabilityManager_initSupervisor_CDADevice(t *testing.T) {
	t.Run("provides an implementation that creates a da device from a zda device", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		capabilities := []da.Capability{0x0001}

		zdaDevice := Device{
			Identifier:   addr,
			Capabilities: capabilities,
			Endpoints:    nil,
		}

		d := s.ComposeDADevice().Compose(zdaDevice)

		assert.Equal(t, addr, d.Identifier())
		assert.Equal(t, capabilities, d.Capabilities())
		assert.Equal(t, zgw, d.Gateway())
	})
}

func TestCapabilityManager_initSupervisor_DeviceLookup(t *testing.T) {
	t.Run("returns false if gateway doesn't match", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr}

		_, ok := s.DeviceLookup().ByDA(daDevice)
		assert.False(t, ok)
	})

	t.Run("returns false if gateway does match, but isn't found in the node table", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr, DeviceGateway: zgw}

		_, ok := s.DeviceLookup().ByDA(daDevice)
		assert.False(t, ok)
	})

	t.Run("returns true if gateway does match and is found, device details match", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr, DeviceGateway: zgw}

		iN, _ := m.nodeTable.createNode(addr.IEEEAddress)
		iD, _ := m.nodeTable.createDevice(addr)
		iD.capabilities = []da.Capability{0x0001}

		iN.endpoints = []zigbee.Endpoint{0x01, 0x02}
		iN.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint: 0x01,
			},
			0x02: {
				Endpoint: 0x02,
			},
		}

		iD.endpoints = []zigbee.Endpoint{0x02}

		expectedEndpoints := map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x02: {
				Endpoint: 0x02,
			},
		}

		d, ok := s.DeviceLookup().ByDA(daDevice)
		assert.True(t, ok)

		assert.Equal(t, iD.capabilities, d.Capabilities)
		assert.Equal(t, addr, d.Identifier)
		assert.Equal(t, expectedEndpoints, d.Endpoints)
	})
}

func TestCapabilityManager_initSupervisor_Poller(t *testing.T) {
	t.Run("calls add on the parent poller with identifier", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		cFn := s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			return true
		})

		assert.NotNil(t, cFn)
	})

	t.Run("when capability provided function is called a populated zda is provided", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		called := false

		_, _, iDev := generateNodeTableWithData(1)

		s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			assert.Equal(t, iDev[0].generateIdentifier(), device.Identifier)
			called = true
			return true
		})

		wrappedFn, ok := mockPoller.Calls[0].Arguments[2].(func(context.Context, *internalDevice) bool)
		assert.True(t, ok)

		ret := wrappedFn(context.TODO(), iDev[0])

		assert.True(t, called)
		assert.True(t, ret)
	})

	t.Run("when poller is cancelled the wrapper returns false without calling the wrapped function", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		_, _, iDev := generateNodeTableWithData(1)

		cFn := s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			t.Fatalf("should not have run wrapper")
			return true
		})

		wrappedFn, ok := mockPoller.Calls[0].Arguments[2].(func(context.Context, *internalDevice) bool)
		assert.True(t, ok)

		cFn()
		ret := wrappedFn(context.TODO(), iDev[0])

		assert.False(t, ret)
	})
}

func TestCapabilityManager_deviceAddedCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("AddedDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceAddedCallback(ctx, internalDeviceAdded{device: device})
		assert.NoError(t, err)

	})
}

func TestCapabilityManager_deviceRemoveCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("RemovedDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceRemovedCallback(ctx, internalDeviceRemoved{device: device})
		assert.NoError(t, err)

	})
}

func TestCapabilityManager_deviceEnumeratedCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("EnumerateDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceEnumeratedCallback(ctx, internalDeviceEnumeration{device: device})
		assert.NoError(t, err)

	})
}
