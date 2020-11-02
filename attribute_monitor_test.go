package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestZclAttributeMonitor_Init(t *testing.T) {
	t.Run("ZCL is correctly attached", func(t *testing.T) {
		mockZCL := &MockZCL{}
		defer mockZCL.AssertExpectations(t)

		attMon := zclAttributeMonitor{zcl: mockZCL}

		mockZCL.On("Listen", mock.AnythingOfType("zda.ZCLFilter"), mock.AnythingOfType("zda.ZCLCallback"))

		attMon.Init()
	})
}

func TestZclAttributeMonitor_zclFilter(t *testing.T) {
	t.Run("returns true if message is ReportAttributes and ClusterID matches", func(t *testing.T) {
		c := zigbee.ClusterID(0x1234)

		attMon := zclAttributeMonitor{clusterID: c}

		matched := attMon.zclFilter(zigbee.IEEEAddress(0), zigbee.ApplicationMessage{}, zcl.Message{
			ClusterID: c,
			Command:   &global.ReportAttributes{},
		})

		assert.True(t, matched)
	})

	t.Run("returns false if message is ReportAttributes, but ClusterID does not match", func(t *testing.T) {
		c := zigbee.ClusterID(0x1234)

		attMon := zclAttributeMonitor{clusterID: c}

		matched := attMon.zclFilter(zigbee.IEEEAddress(0), zigbee.ApplicationMessage{}, zcl.Message{
			ClusterID: zigbee.ClusterID(0),
			Command:   &global.ReportAttributes{},
		})

		assert.False(t, matched)
	})

	t.Run("returns false if message is not ReportAttributes, but ClusterID matches", func(t *testing.T) {
		c := zigbee.ClusterID(0x1234)

		attMon := zclAttributeMonitor{clusterID: c}

		matched := attMon.zclFilter(zigbee.IEEEAddress(0), zigbee.ApplicationMessage{}, zcl.Message{
			ClusterID: c,
			Command:   &global.ReadAttributes{},
		})

		assert.False(t, matched)
	})
}

func TestZclAttributeMonitor_zclMessage(t *testing.T) {
	t.Run("does not callback if device does not have capability", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		d := Device{
			Capabilities: []da.Capability{},
		}

		called := false

		cb := func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue) {
			called = true
		}

		attMon := zclAttributeMonitor{callback: cb, capability: testCap}

		attMon.zclMessage(d, zcl.Message{})

		assert.False(t, called)
	})

	t.Run("does not callback if not a report attribute", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		d := Device{
			Capabilities: []da.Capability{testCap.Capability()},
		}

		called := false

		cb := func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue) {
			called = true
		}

		attMon := zclAttributeMonitor{callback: cb, capability: testCap}

		attMon.zclMessage(d, zcl.Message{
			Command: &global.DiscoverAttributes{},
		})

		assert.False(t, called)
	})

	t.Run("does not callback if wanted attribute does not exist", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		d := Device{
			Capabilities: []da.Capability{testCap.Capability()},
		}

		called := false

		cb := func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue) {
			called = true
		}

		attributeId := zcl.AttributeID(1)

		attMon := zclAttributeMonitor{callback: cb, capability: testCap, attributeID: attributeId}

		attMon.zclMessage(d, zcl.Message{
			Command: &global.ReportAttributes{
				Records: []global.ReportAttributesRecord{},
			},
		})

		assert.False(t, called)
	})

	t.Run("does not callback if wanted attribute type does not match", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		d := Device{
			Capabilities: []da.Capability{testCap.Capability()},
		}

		called := false

		cb := func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue) {
			called = true
		}

		attributeId := zcl.AttributeID(1)
		attributeType := zcl.TypeSignedInt16

		attMon := zclAttributeMonitor{callback: cb, capability: testCap, attributeID: attributeId, attributeDataType: attributeType}

		attMon.zclMessage(d, zcl.Message{
			Command: &global.ReportAttributes{
				Records: []global.ReportAttributesRecord{
					{
						Identifier: attributeId,
						DataTypeValue: &zcl.AttributeDataTypeValue{
							DataType: 0,
							Value:    nil,
						},
					},
				},
			},
		})

		assert.False(t, called)
	})

	t.Run("does callback if all checks pass", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		d := Device{
			Capabilities: []da.Capability{testCap.Capability()},
		}

		called := false
		wantedValue := "wanted"

		cb := func(_ Device, _ zcl.AttributeID, a zcl.AttributeDataTypeValue) {
			called = true
			assert.Equal(t, wantedValue, a.Value.(string))
		}

		attributeId := zcl.AttributeID(1)
		attributeType := zcl.TypeStringCharacter8

		attMon := zclAttributeMonitor{callback: cb, capability: testCap, attributeID: attributeId, attributeDataType: attributeType}

		attMon.zclMessage(d, zcl.Message{
			Command: &global.ReportAttributes{
				Records: []global.ReportAttributesRecord{
					{
						Identifier: attributeId,
						DataTypeValue: &zcl.AttributeDataTypeValue{
							DataType: attributeType,
							Value:    wantedValue,
						},
					},
				},
			},
		})

		assert.True(t, called)
	})
}

func TestZclAttributeMonitor_Attach(t *testing.T) {
	t.Run("returns false if binding and configure reporting succeed", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		mockZCL := MockZCL{}
		defer mockZCL.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		endpoint := zigbee.Endpoint(0x01)

		device := Device{
			Identifier: addr,
		}

		clusterId := zigbee.ClusterID(0x1111)
		attributeId := zcl.AttributeID(0x2222)
		attributeType := zcl.TypeSignedInt16

		providedDefault := int16(16)

		mockZCL.On("Bind", mock.Anything, device, endpoint, clusterId).Return(nil)
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, clusterId, attributeId, attributeType, uint16(0), uint16(60), providedDefault).Return(nil)

		dc := &DefaultDeviceConfig{}
		attMon := zclAttributeMonitor{clusterID: clusterId, attributeID: attributeId, attributeDataType: attributeType, deviceConfig: dc, zcl: &mockZCL, capability: testCap, deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{}, deviceListMutex: &sync.Mutex{}}

		isPolling, err := attMon.Attach(context.Background(), device, endpoint, providedDefault)
		assert.False(t, isPolling)
		assert.NoError(t, err)
	})

	t.Run("returns true if binding fails and configure reporting succeed", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		mockZCL := MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockPoller := MockPoller{}
		defer mockPoller.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		endpoint := zigbee.Endpoint(0x01)

		device := Device{
			Identifier: addr,
		}

		clusterId := zigbee.ClusterID(0x1111)
		attributeId := zcl.AttributeID(0x2222)
		attributeType := zcl.TypeSignedInt16

		providedDefault := int16(16)

		mockZCL.On("Bind", mock.Anything, device, endpoint, clusterId).Return(errors.New("someerror"))
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, clusterId, attributeId, attributeType, uint16(0), uint16(60), providedDefault).Return(nil)

		ret := func() {}
		mockPoller.On("Add", device, 5*time.Second, mock.Anything).Return(ret)

		dc := &DefaultDeviceConfig{}
		attMon := zclAttributeMonitor{clusterID: clusterId, attributeID: attributeId, attributeDataType: attributeType, deviceConfig: dc, poller: &mockPoller, zcl: &mockZCL, capability: testCap, deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{}, deviceListMutex: &sync.Mutex{}}

		isPolling, err := attMon.Attach(context.Background(), device, endpoint, providedDefault)
		assert.Contains(t, attMon.deviceList, device.Identifier)
		assert.NotNil(t, attMon.deviceList[device.Identifier].pollerCancel)
		assert.True(t, isPolling)
		assert.NoError(t, err)
	})

	t.Run("returns true if binding succeeds and configure reporting fails", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		mockZCL := MockZCL{}
		defer mockZCL.AssertExpectations(t)

		mockPoller := MockPoller{}
		defer mockPoller.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		endpoint := zigbee.Endpoint(0x01)

		device := Device{
			Identifier: addr,
		}

		clusterId := zigbee.ClusterID(0x1111)
		attributeId := zcl.AttributeID(0x2222)
		attributeType := zcl.TypeSignedInt16

		providedDefault := int16(16)

		mockZCL.On("Bind", mock.Anything, device, endpoint, clusterId).Return(nil)
		mockZCL.On("ConfigureReporting", mock.Anything, device, endpoint, clusterId, attributeId, attributeType, uint16(0), uint16(60), providedDefault).Return(errors.New("someerror"))

		ret := func() {}
		mockPoller.On("Add", device, 5*time.Second, mock.Anything).Return(ret)

		dc := &DefaultDeviceConfig{}
		attMon := zclAttributeMonitor{clusterID: clusterId, attributeID: attributeId, attributeDataType: attributeType, deviceConfig: dc, poller: &mockPoller, zcl: &mockZCL, capability: testCap, deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{}, deviceListMutex: &sync.Mutex{}}

		isPolling, err := attMon.Attach(context.Background(), device, endpoint, providedDefault)
		assert.Contains(t, attMon.deviceList, device.Identifier)
		assert.NotNil(t, attMon.deviceList[device.Identifier].pollerCancel)
		assert.True(t, isPolling)
		assert.NoError(t, err)
	})
}

func TestZclAttributeMonitor_Load(t *testing.T) {
	t.Run("configures polling if required", func(t *testing.T) {
		testCap := &TestPersistentCapability{}

		mockPoller := MockPoller{}
		defer mockPoller.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := Device{
			Identifier: addr,
		}

		endpoint := zigbee.Endpoint(2)

		ret := func() {}
		mockPoller.On("Add", device, 5*time.Second, mock.Anything).Return(ret)

		dc := &DefaultDeviceConfig{}
		attMon := zclAttributeMonitor{capability: testCap, poller: &mockPoller, deviceConfig: dc, deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{}, deviceListMutex: &sync.Mutex{}}

		attMon.Reattach(context.Background(), device, endpoint, true)
		assert.Contains(t, attMon.deviceList, device.Identifier)
		assert.Equal(t, endpoint, attMon.deviceList[device.Identifier].endpoint)
		assert.NotNil(t, attMon.deviceList[device.Identifier].pollerCancel)
	})
}

func TestZclAttributeMonitor_Detach(t *testing.T) {
	t.Run("if a device has a registered cancel function, it is called and deleted", func(t *testing.T) {
		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := Device{
			Identifier: addr,
		}

		called := false

		testFunc := func() {
			called = true
		}

		attMon := zclAttributeMonitor{deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{device.Identifier: {pollerCancel: testFunc}}, deviceListMutex: &sync.Mutex{}}
		attMon.Detach(context.Background(), device)

		assert.True(t, called)
		assert.NotContains(t, attMon.deviceList, device.Identifier)
	})
}

func TestZclAttributeMonitor_actualPollDevice(t *testing.T) {
	t.Run("returns false if device identifier is not in device list", func(t *testing.T) {
		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := Device{
			Identifier: addr,
		}

		attMon := zclAttributeMonitor{deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{}, deviceListMutex: &sync.Mutex{}}

		assert.False(t, attMon.actualPollDevice(context.Background(), device))
	})

	t.Run("returns true if device identifier is in device list, calls the callback if attribute is included and suceeded", func(t *testing.T) {
		addr := IEEEAddressWithSubIdentifier{
			IEEEAddress:   zigbee.GenerateLocalAdministeredIEEEAddress(),
			SubIdentifier: 0x01,
		}

		device := Device{
			Identifier: addr,
		}

		mockZCL := MockZCL{}
		defer mockZCL.AssertExpectations(t)

		endpoint := zigbee.Endpoint(1)
		cluster := zigbee.ClusterID(0x2222)
		attributeId := zcl.AttributeID(0x0002)
		attributeType := zcl.TypeSignedInt16

		returnedValue := int64(32)

		mockZCL.On("ReadAttributes", mock.Anything, device, endpoint, cluster, []zcl.AttributeID{attributeId}).
			Return(map[zcl.AttributeID]global.ReadAttributeResponseRecord{
				attributeId: {
					Identifier: attributeId,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: attributeType,
						Value:    returnedValue,
					},
				},
			}, nil)

		called := false

		callback := func(d Device, a zcl.AttributeID, val zcl.AttributeDataTypeValue) {
			called = true

			assert.Equal(t, device, d)
			assert.Equal(t, attributeId, a)
			assert.Equal(t, attributeType, val.DataType)
			assert.Equal(t, returnedValue, val.Value)
		}

		attMon := zclAttributeMonitor{callback: callback, zcl: &mockZCL, clusterID: cluster, attributeID: attributeId, attributeDataType: attributeType, deviceList: map[IEEEAddressWithSubIdentifier]monitorDevice{addr: {endpoint: endpoint}}, deviceListMutex: &sync.Mutex{}}

		assert.True(t, attMon.actualPollDevice(context.Background(), device))
		assert.True(t, called)
	})
}

type MockZCL struct {
	mock.Mock
}

func (m *MockZCL) ReadAttributes(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, d, e, c, a)
	return args.Get(0).(map[zcl.AttributeID]global.ReadAttributeResponseRecord), args.Error(1)
}

func (m *MockZCL) WriteAttributes(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a map[zcl.AttributeID]zcl.AttributeDataTypeValue) (map[zcl.AttributeID]global.WriteAttributesResponseRecord, error) {
	args := m.Called(ctx, d, e, c, a)
	return args.Get(0).(map[zcl.AttributeID]global.WriteAttributesResponseRecord), args.Error(1)
}

func (m *MockZCL) Bind(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID) error {
	args := m.Called(ctx, d, e, c)
	return args.Error(0)
}

func (m *MockZCL) ConfigureReporting(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, min uint16, max uint16, chg interface{}) error {
	args := m.Called(ctx, d, e, c, a, dt, min, max, chg)
	return args.Error(0)
}

func (m *MockZCL) WaitForMessage(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, i zcl.CommandIdentifier) (zcl.Message, error) {
	args := m.Called(ctx, d, e, c, i)
	return args.Get(0).(zcl.Message), args.Error(1)
}

func (m *MockZCL) Listen(f ZCLFilter, c ZCLCallback) {
	m.Called(f, c)
}

func (m *MockZCL) RegisterCommandLibrary(cl ZCLCommandLibrary) {
	m.Called(cl)
}

func (m *MockZCL) SendCommand(ctx context.Context, d Device, e zigbee.Endpoint, c zigbee.ClusterID, cmd interface{}) error {
	args := m.Called(ctx, d, e, c, cmd)
	return args.Error(0)
}

type MockPoller struct {
	mock.Mock
}

func (m *MockPoller) Add(d Device, t time.Duration, f func(context.Context, Device) bool) func() {
	args := m.Called(d, t, f)
	return args.Get(0).(func())
}

type DefaultConfig struct {
}

func (m DefaultConfig) String(k string, d string) string {
	return d
}

func (m DefaultConfig) Int(k string, d int) int {
	return d
}

func (m DefaultConfig) Float(k string, d float64) float64 {
	return d
}

func (m DefaultConfig) Bool(k string, d bool) bool {
	return d
}

func (m DefaultConfig) Duration(k string, d time.Duration) time.Duration {
	return d
}

type DefaultDeviceConfig struct {
}

func (m *DefaultDeviceConfig) Get(d Device, k string) Config {
	return DefaultConfig{}
}
