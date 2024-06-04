package temperature_sensor

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence/converter"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/temperature_measurement"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"testing"
	"time"
)

func TestImplementation_BaseFunctions(t *testing.T) {
	t.Run("basic static functions respond correctly", func(t *testing.T) {
		i := NewTemperatureSensor(nil)

		assert.Equal(t, capabilities.TemperatureSensorFlag, i.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.TemperatureSensorFlag], i.Name())
		assert.Equal(t, "ZCLTemperatureSensor", i.ImplName())
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("constructs a new attribute monitor correctly initialising it", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mzi.On("NewAttributeMonitor").Return(mm)

		md := &mocks.MockDevice{}
		defer md.AssertExpectations(t)

		s := memory.New()
		es := s.Section("AttributeMonitor", implcaps.ReadingKey)

		mm.On("Init", es, md, mock.Anything)

		i := NewTemperatureSensor(mzi)
		i.Init(md, s)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads attribute monitor functionality, returning true if successful", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Load", mock.Anything).Return(nil)

		i := NewTemperatureSensor(nil)
		i.am = mm
		attached, err := i.Load(context.TODO())

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("loads attribute monitor functionality, returning false if error", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Load", mock.Anything).Return(io.EOF)

		i := NewTemperatureSensor(nil)
		i.am = mm
		attached, err := i.Load(context.TODO())

		assert.False(t, attached)
		assert.Error(t, err)
	})
}

func TestImplementation_Enumerate(t *testing.T) {
	t.Run("attaches to the attribute monitor", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Attach", mock.Anything, zigbee.Endpoint(0x01), zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, mock.Anything, mock.Anything).Return(nil)

		i := NewTemperatureSensor(nil)
		i.am = mm
		attached, err := i.Enumerate(context.TODO(), make(map[string]any))

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("fails if attach to the attribute monitor fails", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Attach", mock.Anything, zigbee.Endpoint(0x01), zcl.TemperatureMeasurementId, temperature_measurement.MeasuredValue, zcl.TypeSignedInt16, mock.Anything, mock.Anything).Return(io.EOF)

		i := NewTemperatureSensor(nil)
		i.am = mm
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

		i := NewTemperatureSensor(nil)
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
			e, ok := args.Get(0).(capabilities.TemperatureSensorUpdate)
			assert.True(t, ok)
			assert.InEpsilon(t, 293.5, e.State[0].Value, 0.001)
		})

		i := NewTemperatureSensor(mzi)
		i.s = memory.New()

		i.s.Set(implcaps.ReadingKey, 293.4)

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(0, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeSignedInt16,
			Value:    int64(2040),
		})

		temp, _ := i.Reading(context.TODO())
		assert.InEpsilon(t, 293.5, temp[0].Value, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("updates the state correctly, no event if no change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		i := NewTemperatureSensor(mzi)
		i.s = memory.New()

		i.s.Set(implcaps.ReadingKey, 293.5)

		lastUpdated := time.UnixMilli(time.Now().UnixMilli()).Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(0, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeSignedInt16,
			Value:    int64(2040),
		})

		temp, _ := i.Reading(context.TODO())
		assert.InEpsilon(t, 293.5, temp[0].Value, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Equal(t, lct, lastUpdated)
	})
}

func TestImplementation_Reading(t *testing.T) {
	t.Run("returns the current temperature", func(t *testing.T) {
		i := NewTemperatureSensor(nil)
		i.s = memory.New()

		i.s.Set(implcaps.ReadingKey, 240.5)

		d, err := i.Reading(context.TODO())
		assert.NoError(t, err)
		assert.Len(t, d, 1)
		assert.Equal(t, 240.5, d[0].Value)
	})
}

func TestImplementation_LastTimes(t *testing.T) {
	t.Run("returns the last updated and changed times", func(t *testing.T) {
		i := NewTemperatureSensor(nil)
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
