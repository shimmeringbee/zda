package identify

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence/converter"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/identify"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"sync"
	"testing"
	"time"
)

func TestImplementation_BaseFunctions(t *testing.T) {
	t.Run("basic static functions respond correctly", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)
		mzi.On("ZCLRegister", mock.Anything)

		i := NewIdentify(mzi)

		assert.Equal(t, capabilities.IdentifyFlag, i.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.IdentifyFlag], i.Name())
		assert.Equal(t, "ZCLIdentify", i.ImplName())
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("constructs a new attribute monitor correctly initialising it", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)
		mzi.On("ZCLRegister", mock.Anything)

		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mzi.On("NewAttributeMonitor").Return(mm)

		md := &mocks.MockDevice{}
		defer md.AssertExpectations(t)

		s := memory.New()
		es := s.Section("AttributeMonitor", implcaps.ReadingKey)

		mm.On("Init", es, md, mock.Anything)

		i := NewIdentify(mzi)
		i.Init(md, s)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads attribute monitor functionality, returning true if successful", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Load", mock.Anything).Return(nil)

		i := &Implementation{}
		i.am = mm
		i.s = memory.New()

		i.s.Set(implcaps.RemoteEndpointKey, 1)
		i.s.Set(implcaps.ClusterIdKey, 1)

		attached, err := i.Load(context.TODO())

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("loads attribute monitor functionality, returning false if error", func(t *testing.T) {
		i := &Implementation{}
		i.s = memory.New()

		attached, err := i.Load(context.TODO())

		assert.False(t, attached)
		assert.Error(t, err)
	})
}

func TestImplementation_Enumerate(t *testing.T) {
	t.Run("attaches to the attribute monitor, using default attributes", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Attach", mock.Anything, zigbee.Endpoint(0x01), zcl.IdentifyId, identify.IdentifyTime, zcl.TypeUnsignedInt16, mock.Anything, mock.Anything).Return(nil)

		i := &Implementation{}
		i.am = mm
		i.s = memory.New()
		attached, err := i.Enumerate(context.TODO(), make(map[string]any))

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("attaches to the attribute monitor, using overridden attributes", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Attach", mock.Anything, zigbee.Endpoint(0x02), zigbee.ClusterID(0x500), zcl.AttributeID(0x10), zcl.TypeUnsignedInt16, mock.Anything, mock.Anything).Return(nil)

		i := &Implementation{}
		i.am = mm
		i.s = memory.New()

		attributes := map[string]any{
			"ZigbeeEndpoint":                    zigbee.Endpoint(0x02),
			"ZigbeeIdentifyClusterID":           zigbee.ClusterID(0x500),
			"ZigbeeIdentifyDurationAttributeID": zcl.AttributeID(0x10),
		}
		attached, err := i.Enumerate(context.TODO(), attributes)

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("fails if attach to the attribute monitor fails", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Attach", mock.Anything, zigbee.Endpoint(0x01), zcl.IdentifyId, identify.IdentifyTime, zcl.TypeUnsignedInt16, mock.Anything, mock.Anything).Return(io.EOF)

		i := &Implementation{}
		i.am = mm
		i.s = memory.New()
		attached, err := i.Enumerate(context.TODO(), make(map[string]any))

		assert.False(t, attached)
		assert.Error(t, err)
	})
}

func TestImplementation_Detach(t *testing.T) {
	t.Run("detached attribute monitor on detach", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Detach", mock.Anything, true).Return(nil)

		i := &Implementation{}
		i.am = mm

		err := i.Detach(context.TODO(), implcaps.NoLongerEnumerated)
		assert.NoError(t, err)
	})
}

func TestImplementation_update(t *testing.T) {
	t.Run("updates the state correctly, sending even if change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.IdentifyUpdate)
			assert.True(t, ok)
			assert.InDelta(t, 5*time.Second, e.State.Remaining, float64(time.Duration(100)*time.Millisecond))
			assert.True(t, e.State.Identifying)
		})

		i := Implementation{timerMutex: &sync.Mutex{}}
		i.zi = mzi
		i.s = memory.New()

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(0, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(5),
		})

		e, _ := i.Status(context.TODO())
		assert.InDelta(t, 5*time.Second, e.Remaining, float64(time.Duration(100)*time.Millisecond))
		assert.True(t, e.Identifying)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("updates the state correctly, no event if no change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		i := Implementation{}
		i.zi = mzi
		i.s = memory.New()

		converter.Store(i.s, EndTimeKey, time.Now().Add(5*time.Second), converter.TimeEncoder)

		lastUpdated := time.UnixMilli(time.Now().UnixMilli()).Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(0, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(5),
		})

		e, _ := i.Status(context.TODO())
		assert.InDelta(t, 5*time.Second, e.Remaining, float64(time.Duration(100)*time.Millisecond))
		assert.True(t, e.Identifying)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Equal(t, lct, lastUpdated)
	})
}

func TestImplementation_Status(t *testing.T) {
	t.Run("returns the status of identity", func(t *testing.T) {
		i := &Implementation{}
		i.s = memory.New()

		converter.Store(i.s, EndTimeKey, time.Now().Add(5*time.Second), converter.TimeEncoder)

		d, err := i.Status(context.TODO())
		assert.NoError(t, err)

		assert.True(t, d.Identifying)
		assert.Greater(t, d.Remaining, time.Duration(4900)*time.Millisecond)
	})
}

func TestImplementation_LastTimes(t *testing.T) {
	t.Run("returns the last updated and changed times", func(t *testing.T) {
		i := &Implementation{}
		i.s = memory.New()

		changedTime := time.UnixMilli(time.Now().UnixMilli())
		updatedTime := changedTime.Add(5 * time.Minute)

		converter.Store(i.s, implcaps.LastUpdatedKey, updatedTime, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, changedTime, converter.TimeEncoder)

		lct, err := i.LastChangeTime(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, changedTime, lct)

		lut, err := i.LastUpdateTime(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, updatedTime, lut)
	})
}

func TestImplementation_Identify(t *testing.T) {
	t.Run("sends Identify packet to device", func(t *testing.T) {
		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)
		mzi.On("ZCLRegister", mock.Anything)
		mzi.On("ZCLCommunicator").Return(mzc)

		md := &mocks.MockDevice{}
		defer md.AssertExpectations(t)

		i := NewIdentify(mzi)
		i.d = md
		i.s = memory.New()

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		localEndpoint := zigbee.Endpoint(0x01)
		seq := 8

		mzi.On("TransmissionLookup", md, zigbee.ProfileHomeAutomation).Return(ieee, localEndpoint, false, seq)

		i.clusterId = zcl.IdentifyId
		i.remoteEndpoint = 4

		expectedMsg := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: uint8(seq),
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           i.clusterId,
			SourceEndpoint:      localEndpoint,
			DestinationEndpoint: i.remoteEndpoint,
			CommandIdentifier:   identify.IdentifyId,
			Command:             &identify.Identify{IdentifyTime: 5},
		}

		mzc.On("Request", mock.Anything, ieee, false, mock.MatchedBy(func(actualMsg zcl.Message) bool {
			return assert.ObjectsAreEqual(expectedMsg, actualMsg)
		})).Return(nil)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.IdentifyUpdate)
			assert.True(t, ok)
			assert.InDelta(t, 5*time.Second, e.State.Remaining, float64(time.Duration(100)*time.Millisecond))
			assert.True(t, e.State.Identifying)
		})

		now := time.Now()

		err := i.Identify(context.TODO(), 5*time.Second)
		assert.NoError(t, err)

		val, ok := i.s.Int(EndTimeKey)
		assert.True(t, ok)

		endTime := time.UnixMilli(int64(val))
		assert.True(t, endTime.After(now))
	})
}
