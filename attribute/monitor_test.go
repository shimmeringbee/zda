package attribute

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"testing"
	"time"
)

func Test_zclMonitor_Init(t *testing.T) {
	t.Run("constructor and Init sets up struct correctly", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return 0, 0, false, 0
		}

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)

		assert.Equal(t, mzc, z.zclCommunicator)
		assert.Equal(t, mzp, z.nodeBinder)
		assert.NotNil(t, z.transmissionLookup)

		assert.Equal(t, s, z.config)
		assert.Equal(t, d, z.device)
		assert.NotNil(t, z.callback)

		assert.NotNil(t, z.pollerStop)
	})
}

func Test_zclMonitor_Attach(t *testing.T) {
	t.Run("populates structure correctly, no polling or reporting", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)

		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(dd da.Device, _ zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			assert.Equal(t, d, dd)
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Attach(context.Background(), 1, 2, 3, zcl.TypeUnsignedInt8, ReportingConfig{Mode: NeverConfigureReporting}, PollingConfig{Mode: NeverPoll})
		assert.NoError(t, err)

		assert.Equal(t, expectedIeee, z.ieeeAddress)
		assert.Equal(t, zigbee.Endpoint(2), z.localEndpoint)

		assert.Equal(t, zigbee.Endpoint(1), z.remoteEndpoint)
		assert.Equal(t, zigbee.ClusterID(2), z.clusterID)
		assert.Equal(t, zcl.AttributeID(3), z.attributeID)
		assert.Equal(t, zcl.TypeUnsignedInt8, z.attributeDataType)

		remoteEndpointSetting, _ := z.config.Int(RemoteEndpointKey)
		assert.Equal(t, int(z.remoteEndpoint), remoteEndpointSetting)

		clusterIdSetting, _ := z.config.Int(ClusterIdKey)
		assert.Equal(t, int(z.clusterID), clusterIdSetting)

		attributeIdSetting, _ := z.config.Int(AttributeIdKey)
		assert.Equal(t, int(z.attributeID), attributeIdSetting)

		attributeDataTypeSetting, _ := z.config.Int(AttributeDataTypeKey)
		assert.Equal(t, int(z.attributeDataType), attributeDataTypeSetting)

		assert.NotNil(t, z.match)
	})

	t.Run("attach succeeds for reporting only", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		mzp.On("BindNodeToController", mock.Anything, expectedIeee, zigbee.Endpoint(2), zigbee.Endpoint(1), zigbee.ClusterID(2)).Return(nil)
		mzc.On("ConfigureReporting", mock.Anything, expectedIeee, false, zigbee.ClusterID(2), zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(0), zcl.AttributeID(3), zcl.TypeUnsignedInt8, uint16(60), uint16(300), nil).Return(nil)

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		tl := func(dd da.Device, _ zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			assert.Equal(t, d, dd)
			return expectedIeee, 2, false, 0
		}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Attach(context.Background(), 1, 2, 3, zcl.TypeUnsignedInt8, ReportingConfig{Mode: AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: nil}, PollingConfig{Mode: NeverPoll})
		assert.NoError(t, err)

		reportingConfiguredSetting, _ := z.config.Bool(ReportingConfiguredKey)
		assert.True(t, reportingConfiguredSetting)
	})

	t.Run("attach succeeds for reporting fails, polling if failed, polling configured", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		mzp.On("BindNodeToController", mock.Anything, expectedIeee, zigbee.Endpoint(2), zigbee.Endpoint(1), zigbee.ClusterID(2)).Return(io.EOF)

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(dd da.Device, _ zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			assert.Equal(t, d, dd)
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Attach(context.Background(), 1, 2, 3, zcl.TypeUnsignedInt8, ReportingConfig{Mode: AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: nil}, PollingConfig{Mode: PollIfReportingFailed, Interval: time.Minute})
		assert.NoError(t, err)

		reportingConfiguredSetting, _ := z.config.Bool(ReportingConfiguredKey)
		assert.False(t, reportingConfiguredSetting)

		pollingConfiguredSetting, _ := z.config.Bool(PollingConfiguredKey)
		assert.True(t, pollingConfiguredSetting)

		pollingIntervalSetting, _ := z.config.Int(PollingIntervalKey)
		assert.Equal(t, 60000, pollingIntervalSetting)
		assert.NotNil(t, z.ticker)
	})

	t.Run("attach succeeds for reporting succeeds, polling always", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		mzp.On("BindNodeToController", mock.Anything, expectedIeee, zigbee.Endpoint(2), zigbee.Endpoint(1), zigbee.ClusterID(2)).Return(nil)
		mzc.On("ConfigureReporting", mock.Anything, expectedIeee, false, zigbee.ClusterID(2), zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(0), zcl.AttributeID(3), zcl.TypeUnsignedInt8, uint16(60), uint16(300), nil).Return(nil)

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Attach(context.Background(), 1, 2, 3, zcl.TypeUnsignedInt8, ReportingConfig{Mode: AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: nil}, PollingConfig{Mode: AlwaysPoll, Interval: time.Minute})
		assert.NoError(t, err)

		reportingConfiguredSetting, _ := z.config.Bool(ReportingConfiguredKey)
		assert.True(t, reportingConfiguredSetting)

		pollingConfiguredSetting, _ := z.config.Bool(PollingConfiguredKey)
		assert.True(t, pollingConfiguredSetting)

		pollingIntervalSetting, _ := z.config.Int(PollingIntervalKey)
		assert.Equal(t, 60000, pollingIntervalSetting)
		assert.NotNil(t, z.ticker)
	})

	t.Run("attach succeeds for reporting off, polling always", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		s := memory.New()

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Attach(context.Background(), 1, 2, 3, zcl.TypeUnsignedInt8, ReportingConfig{Mode: NeverConfigureReporting}, PollingConfig{Mode: AlwaysPoll, Interval: time.Minute})
		assert.NoError(t, err)

		reportingConfiguredSetting, _ := z.config.Bool(ReportingConfiguredKey)
		assert.False(t, reportingConfiguredSetting)

		pollingConfiguredSetting, _ := z.config.Bool(PollingConfiguredKey)
		assert.True(t, pollingConfiguredSetting)

		pollingIntervalSetting, _ := z.config.Int(PollingIntervalKey)
		assert.Equal(t, 60000, pollingIntervalSetting)
		assert.NotNil(t, z.ticker)
	})
}

func Test_zclMonitor_Load(t *testing.T) {
	t.Run("load without starting polling", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		s := memory.New()
		s.Set(RemoteEndpointKey, 1)
		s.Set(ClusterIdKey, 2)
		s.Set(AttributeIdKey, 3)
		s.Set(AttributeDataTypeKey, int(zcl.TypeUnsignedInt8))

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Load(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, expectedIeee, z.ieeeAddress)
		assert.Equal(t, zigbee.Endpoint(2), z.localEndpoint)

		assert.Equal(t, zigbee.Endpoint(1), z.remoteEndpoint)
		assert.Equal(t, zigbee.ClusterID(2), z.clusterID)
		assert.Equal(t, zcl.AttributeID(3), z.attributeID)
		assert.Equal(t, zcl.TypeUnsignedInt8, z.attributeDataType)

		assert.Nil(t, z.ticker)
	})

	t.Run("load starting polling", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		s := memory.New()
		s.Set(RemoteEndpointKey, 1)
		s.Set(ClusterIdKey, 2)
		s.Set(AttributeIdKey, 3)
		s.Set(AttributeDataTypeKey, int(zcl.TypeUnsignedInt8))
		s.Set(PollingConfiguredKey, true)
		s.Set(PollingIntervalKey, time.Minute.Milliseconds())

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		defer z.Detach(context.Background(), false)

		err := z.Load(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, expectedIeee, z.ieeeAddress)
		assert.Equal(t, zigbee.Endpoint(2), z.localEndpoint)

		assert.Equal(t, zigbee.Endpoint(1), z.remoteEndpoint)
		assert.Equal(t, zigbee.ClusterID(2), z.clusterID)
		assert.Equal(t, zcl.AttributeID(3), z.attributeID)
		assert.Equal(t, zcl.TypeUnsignedInt8, z.attributeDataType)

		assert.NotNil(t, z.ticker)
	})
}

func Test_zclMonitor_Detach(t *testing.T) {
	t.Run("detach with unconfigure", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)
		mzc.On("RegisterMatch", mock.Anything)
		mzc.On("UnregisterMatch", mock.Anything)

		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()
		mzc.On("ConfigureReporting", mock.Anything, expectedIeee, false, zigbee.ClusterID(2), zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(0), zcl.AttributeID(3), zcl.TypeUnsignedInt8, uint16(0xffff), uint16(0x0000), nil).Return(nil)

		mzp := &zigbee.MockProvider{}
		defer mzp.AssertExpectations(t)

		s := memory.New()
		s.Set(RemoteEndpointKey, 1)
		s.Set(ClusterIdKey, 2)
		s.Set(AttributeIdKey, 3)
		s.Set(AttributeDataTypeKey, int(zcl.TypeUnsignedInt8))
		s.Set(PollingConfiguredKey, true)
		s.Set(ReportingConfiguredKey, true)
		s.Set(PollingIntervalKey, time.Minute.Milliseconds())

		d := &mocks.MockDevice{}
		defer d.AssertExpectations(t)
		d.On("Identifier").Return(zigbee.GenerateLocalAdministeredIEEEAddress())

		tl := func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		cb := func(zcl.AttributeID, zcl.AttributeDataTypeValue) {}

		z := NewMonitor(mzc, mzp, tl, logwrap.New(discard.Discard())).(*zclMonitor)
		z.Init(s, d, cb)
		err := z.Load(context.Background())
		assert.NoError(t, err)

		err = z.Detach(context.Background(), true)
		assert.NoError(t, err)

		_, reportingPresent := z.config.Bool(ReportingConfiguredKey)
		assert.False(t, reportingPresent)

		_, pollingPresent := z.config.Bool(PollingConfiguredKey)
		assert.False(t, pollingPresent)

		assert.Nil(t, z.ticker)
	})
}

func Test_zclMonitor_poller(t *testing.T) {
	t.Run("polls device for data when requested", func(t *testing.T) {
		expectedIeee := zigbee.GenerateLocalAdministeredIEEEAddress()

		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		z := &zclMonitor{
			pollerStop:      make(chan struct{}, 1),
			ticker:          time.NewTicker(time.Millisecond),
			ieeeAddress:     expectedIeee,
			clusterID:       1,
			localEndpoint:   2,
			remoteEndpoint:  3,
			attributeID:     4,
			zclCommunicator: mzc,
		}

		mzc.On("ReadAttributes", mock.Anything, expectedIeee, false, z.clusterID, zigbee.NoManufacturer, z.localEndpoint, z.remoteEndpoint, uint8(0), []zcl.AttributeID{4}).Return([]global.ReadAttributeResponseRecord{}, nil)

		defer func() {
			z.pollerStop <- struct{}{}
		}()

		z.transmissionLookup = func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
			return expectedIeee, 2, false, 0
		}

		go z.poller(context.Background())

		time.Sleep(25 * time.Millisecond)
	})
}

func Test_zclMonitor_zclFilter(t *testing.T) {
	t.Run("returns true if everything matches", func(t *testing.T) {
		z := zclMonitor{}
		z.ieeeAddress = zigbee.GenerateLocalAdministeredIEEEAddress()
		z.localEndpoint = 1
		z.remoteEndpoint = 2

		match := z.zclFilter(z.ieeeAddress, zigbee.ApplicationMessage{}, zcl.Message{
			SourceEndpoint:      z.remoteEndpoint,
			DestinationEndpoint: z.localEndpoint,
			Direction:           zcl.ServerToClient,
		})

		assert.True(t, match)
	})

	t.Run("returns false if ieee doesn't match", func(t *testing.T) {
		z := zclMonitor{}
		z.ieeeAddress = zigbee.GenerateLocalAdministeredIEEEAddress()
		z.localEndpoint = 1
		z.remoteEndpoint = 2

		match := z.zclFilter(zigbee.GenerateLocalAdministeredIEEEAddress(), zigbee.ApplicationMessage{}, zcl.Message{
			SourceEndpoint:      z.remoteEndpoint,
			DestinationEndpoint: z.localEndpoint,
			Direction:           zcl.ServerToClient,
		})

		assert.False(t, match)
	})

	t.Run("returns false if source endpoint doesn't match", func(t *testing.T) {
		z := zclMonitor{}
		z.ieeeAddress = zigbee.GenerateLocalAdministeredIEEEAddress()
		z.localEndpoint = 1
		z.remoteEndpoint = 2

		match := z.zclFilter(z.ieeeAddress, zigbee.ApplicationMessage{}, zcl.Message{
			SourceEndpoint:      99,
			DestinationEndpoint: z.localEndpoint,
			Direction:           zcl.ServerToClient,
		})

		assert.False(t, match)
	})

	t.Run("returns false if destination endpoint doesn't match", func(t *testing.T) {
		z := zclMonitor{}
		z.ieeeAddress = zigbee.GenerateLocalAdministeredIEEEAddress()
		z.localEndpoint = 1
		z.remoteEndpoint = 2

		match := z.zclFilter(z.ieeeAddress, zigbee.ApplicationMessage{}, zcl.Message{
			SourceEndpoint:      z.remoteEndpoint,
			DestinationEndpoint: 99,
			Direction:           zcl.ServerToClient,
		})

		assert.False(t, match)
	})

	t.Run("returns false if direction doesn't match", func(t *testing.T) {
		z := zclMonitor{}
		z.ieeeAddress = zigbee.GenerateLocalAdministeredIEEEAddress()
		z.localEndpoint = 1
		z.remoteEndpoint = 2

		match := z.zclFilter(z.ieeeAddress, zigbee.ApplicationMessage{}, zcl.Message{
			SourceEndpoint:      z.remoteEndpoint,
			DestinationEndpoint: z.localEndpoint,
			Direction:           zcl.ClientToServer,
		})

		assert.False(t, match)
	})
}

func Test_zclMonitor_zclMessage(t *testing.T) {
	t.Run("callback activated for matching ReadAttribute with success state", func(t *testing.T) {
		called := false

		value := &zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    nil,
		}

		z := zclMonitor{}
		z.attributeID = 1
		z.attributeDataType = zcl.TypeUnsignedInt16
		z.callback = func(id zcl.AttributeID, cbValue zcl.AttributeDataTypeValue) {
			called = true

			assert.Equal(t, z.attributeID, id)
			assert.Equal(t, *value, cbValue)
		}

		z.zclMessage(communicator.MessageWithSource{
			Message: zcl.Message{
				Command: &global.ReadAttributesResponse{
					Records: []global.ReadAttributeResponseRecord{
						{
							Identifier:    z.attributeID,
							Status:        0, // Is success
							DataTypeValue: value,
						},
					},
				},
			},
		})

		assert.True(t, called)
	})

	t.Run("callback not used for matching ReadAttribute with failure state", func(t *testing.T) {
		value := &zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    nil,
		}

		z := zclMonitor{}
		z.attributeID = 1
		z.attributeDataType = zcl.TypeUnsignedInt16
		z.callback = func(id zcl.AttributeID, cbValue zcl.AttributeDataTypeValue) {
			assert.Fail(t, "callback called incorrectly")
		}

		z.zclMessage(communicator.MessageWithSource{
			Message: zcl.Message{
				Command: &global.ReadAttributesResponse{
					Records: []global.ReadAttributeResponseRecord{
						{
							Identifier:    z.attributeID,
							Status:        1, // Is Failure
							DataTypeValue: value,
						},
					},
				},
			},
		})
	})

	t.Run("callback activated for matching ReportAttributes", func(t *testing.T) {
		called := false

		value := &zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    nil,
		}

		z := zclMonitor{}
		z.attributeID = 1
		z.attributeDataType = zcl.TypeUnsignedInt16
		z.callback = func(id zcl.AttributeID, cbValue zcl.AttributeDataTypeValue) {
			called = true

			assert.Equal(t, z.attributeID, id)
			assert.Equal(t, *value, cbValue)
		}

		z.zclMessage(communicator.MessageWithSource{
			Message: zcl.Message{
				Command: &global.ReportAttributes{
					Records: []global.ReportAttributesRecord{
						{
							Identifier:    z.attributeID,
							DataTypeValue: value,
						},
					},
				},
			},
		})

		assert.True(t, called)
	})
}
