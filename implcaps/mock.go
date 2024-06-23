package implcaps

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/mock"
)

type MockZDAInterface struct {
	mock.Mock
}

func (m *MockZDAInterface) NodeBinder() zigbee.NodeBinder {
	return m.Called().Get(0).(zigbee.NodeBinder)
}

func (m *MockZDAInterface) Logger() logwrap.Logger {
	return m.Called().Get(0).(logwrap.Logger)
}

func (m *MockZDAInterface) ZCLRegister(f func(*zcl.CommandRegistry)) {
	m.Called(f)
}

func (m *MockZDAInterface) TransmissionLookup(device da.Device, id zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
	args := m.Called(device, id)
	return args.Get(0).(zigbee.IEEEAddress), args.Get(1).(zigbee.Endpoint), args.Bool(2), uint8(args.Int(3))
}

func (m *MockZDAInterface) ZCLCommunicator() communicator.Communicator {
	return m.Called().Get(0).(communicator.Communicator)
}

func (m *MockZDAInterface) NewAttributeMonitor() attribute.Monitor {
	return m.Called().Get(0).(attribute.Monitor)
}

func (m *MockZDAInterface) SendEvent(a any) {
	m.Called(a)
}

var _ ZDAInterface = (*MockZDAInterface)(nil)
