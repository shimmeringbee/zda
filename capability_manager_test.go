package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
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
		mC.On("Name").Return(kN)

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
		mC.On("Name").Return(kN)

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
		mC.On("Name").Return(kN)

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
		mC.On("Name").Return(kN)

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
		mC.On("Name").Return(kN)
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
		mC.On("Name").Return(kN)
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
		mC.On("Name").Return(kN)
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

func (m *mockCapability) Name() string {
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
