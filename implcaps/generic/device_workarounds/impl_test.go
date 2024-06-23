package device_workaround

import (
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zda/implcaps"
	mocks2 "github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestImplementation_BaseFunctions(t *testing.T) {
	t.Run("basic static functions respond correctly", func(t *testing.T) {
		i := NewDeviceWorkaround(nil)

		assert.Equal(t, capabilities.DeviceWorkaroundsFlag, i.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.DeviceWorkaroundsFlag], i.Name())
		assert.Equal(t, "GenericDeviceWorkarounds", i.ImplName())
	})
}

func TestImplementation_EnableZCLReportingKeepAlive(t *testing.T) {
	t.Run("configures reporting for ZCLVersion on Enumeration", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mnb := &zigbee.MockProvider{}
		defer mnb.AssertExpectations(t)

		md := &mocks.MockDevice{}
		defer md.AssertExpectations(t)

		mzc := &mocks2.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		mzi.On("NodeBinder").Return(mnb)
		mzi.On("ZCLCommunicator").Return(mzc)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		mzi.On("TransmissionLookup", md, zigbee.ProfileHomeAutomation).Return(ieee, zigbee.Endpoint(2), false, 4)

		mnb.On("BindNodeToController", mock.Anything, ieee, zigbee.Endpoint(2), zigbee.Endpoint(3), zcl.BasicId).Return(nil)

		mzc.On("ConfigureReporting", mock.Anything, ieee, false, zcl.BasicId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(3), uint8(4), basic.ZCLVersion, zcl.TypeUnsignedInt8, uint16(60), uint16(240), 0).Return(nil)

		s := memory.New()
		i := NewDeviceWorkaround(mzi)
		i.Init(md, s)

		attached, err := i.Enumerate(nil, map[string]any{"EnableZCLReportingKeepAlive": true, "ZigbeeEndpoint": zigbee.Endpoint(3)})
		assert.NoError(t, err)
		assert.True(t, attached)

		ws := s.Section("Workarounds")
		assert.Contains(t, ws.SectionKeys(), "ZCLReportingKeepAlive")
	})
}
