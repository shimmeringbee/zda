package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestCapabilityManager_Add(t *testing.T) {
	t.Run("makes it available in the capability by flag store", func(t *testing.T) {
		m := NewCapabilityManager()

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
		m := NewCapabilityManager()

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)

		m.Add(mC)
		actualCapabilities := m.PersistingCapabilities()

		assert.Equal(t, mC, actualCapabilities[kN])
	})
}

func TestCapabilityManager_Init(t *testing.T) {
	t.Run("inits any added capabilities that support it", func(t *testing.T) {
		m := NewCapabilityManager()

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("KeyName").Return(kN)
		mC.On("Init", mock.Anything)

		m.Add(mC)
		m.Init()
	})
}

func TestCapabilityManager_StartStop(t *testing.T) {
	t.Run("starts any added capabilities that support it", func(t *testing.T) {
		m := NewCapabilityManager()

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
		m := NewCapabilityManager()

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
