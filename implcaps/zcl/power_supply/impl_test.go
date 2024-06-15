package power_suply

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	mocks2 "github.com/shimmeringbee/da/mocks"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/persistence/converter"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
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
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)

		assert.Equal(t, capabilities.PowerSupplyFlag, i.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.PowerSupplyFlag], i.Name())
		assert.Equal(t, "ZCLPowerSupply", i.ImplName())
	})
}

func TestImplementation_Init(t *testing.T) {
	t.Run("constructs a new attribute monitor correctly initialising it", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mzi.On("NewAttributeMonitor").Return(mm)

		md := &mocks2.MockDevice{}
		defer md.AssertExpectations(t)

		s := memory.New()
		es := s.Section("AttributeMonitor", implcaps.ReadingKey)

		mm.On("Init", es, md, mock.Anything)

		i := NewPowerSupply(mzi)
		i.Init(md, s)
	})
}

func TestImplementation_Load(t *testing.T) {
	t.Run("loads attribute monitor functionality, returning true if successful", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Load", mock.Anything).Return(nil)

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.s.Set(MainsVoltagePresentKey, true)
		i.mainsVoltageMonitor = mm
		attached, err := i.Load(context.TODO())

		assert.True(t, attached)
		assert.NoError(t, err)
	})

	t.Run("loads attribute monitor functionality, returning false if error", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Load", mock.Anything).Return(io.EOF)

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.s.Set(MainsVoltagePresentKey, true)
		i.mainsVoltageMonitor = mm
		attached, err := i.Load(context.TODO())

		assert.False(t, attached)
		assert.Error(t, err)
	})
}

func TestImplementation_Enumerate(t *testing.T) {
	t.Run("performed basic power source enumeration, if cluster present", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		ieeeAddress := zigbee.GenerateLocalAdministeredIEEEAddress()
		mzi.On("TransmissionLookup", mock.Anything, zigbee.ProfileHomeAutomation).Return(ieeeAddress, zigbee.Endpoint(2), true, 4)

		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		mzi.On("ZCLCommunicator").Return(mzc)

		mzc.On("ReadAttributes", mock.Anything, ieeeAddress, true, zcl.BasicId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(4), []zcl.AttributeID{basic.PowerSource}).
			Return([]global.ReadAttributeResponseRecord{{Identifier: basic.PowerSource, Status: 0, DataTypeValue: &zcl.AttributeDataTypeValue{Value: uint8(0x81)}}}, nil)

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		attached, err := i.Enumerate(context.Background(), map[string]any{"ZigbeeEndpoint": 1, "ZigbeeBasicClusterPresent": true})
		assert.NoError(t, err)
		assert.True(t, attached)

		assert.True(t, i.mainsPresent)
		assert.True(t, i.batteryPresent[0])

		v, _ := i.s.Bool(MainsPresentKey)
		assert.True(t, v)

		vp, _ := i.s.Bool(BatteryPresent(0))
		assert.True(t, vp)
	})

	t.Run("performed power configuration enumeration, if cluster present", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		ieeeAddress := zigbee.GenerateLocalAdministeredIEEEAddress()
		mzi.On("TransmissionLookup", mock.Anything, zigbee.ProfileHomeAutomation).Return(ieeeAddress, zigbee.Endpoint(2), true, 4)

		mzc := &mocks.MockZCLCommunicator{}
		defer mzc.AssertExpectations(t)

		mzi.On("ZCLCommunicator").Return(mzc)

		mam := &attribute.MockMonitor{}
		defer mam.AssertExpectations(t)

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsVoltageMonitor = mam
		i.mainsFrequencyMonitor = mam
		i.batteryVoltageMonitor[0] = mam
		i.batteryPercentageMonitor[0] = mam

		mzc.On("ReadAttributes", mock.Anything, ieeeAddress, true, zcl.PowerConfigurationId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(4), []zcl.AttributeID{
			power_configuration.MainsVoltage,
			power_configuration.MainsFrequency,
		}).Return([]global.ReadAttributeResponseRecord{{Identifier: power_configuration.MainsVoltage, Status: 0}, {Identifier: power_configuration.MainsFrequency, Status: 0}}, nil)

		mam.On("Attach", mock.Anything, zigbee.Endpoint(1), zcl.PowerConfigurationId, power_configuration.MainsVoltage, zcl.TypeUnsignedInt16, mock.Anything, mock.Anything).Return(nil)
		mam.On("Attach", mock.Anything, zigbee.Endpoint(1), zcl.PowerConfigurationId, power_configuration.MainsFrequency, zcl.TypeUnsignedInt8, mock.Anything, mock.Anything).Return(nil)

		mzc.On("ReadAttributes", mock.Anything, ieeeAddress, true, zcl.PowerConfigurationId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(4), []zcl.AttributeID{
			power_configuration.BatteryVoltage,
			power_configuration.BatteryPercentageRemaining,
		}).Return([]global.ReadAttributeResponseRecord{{Identifier: power_configuration.BatteryVoltage, Status: 0}, {Identifier: power_configuration.BatteryPercentageRemaining, Status: 0}}, nil)

		mam.On("Attach", mock.Anything, zigbee.Endpoint(1), zcl.PowerConfigurationId, power_configuration.BatteryVoltage, zcl.TypeUnsignedInt8, mock.Anything, mock.Anything).Return(nil)
		mam.On("Attach", mock.Anything, zigbee.Endpoint(1), zcl.PowerConfigurationId, power_configuration.BatteryPercentageRemaining, zcl.TypeUnsignedInt8, mock.Anything, mock.Anything).Return(nil)

		mzc.On("ReadAttributes", mock.Anything, ieeeAddress, true, zcl.PowerConfigurationId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(4), []zcl.AttributeID{
			power_configuration.BatterySource2Voltage,
			power_configuration.BatterySource2PercentageRemaining,
		}).Return([]global.ReadAttributeResponseRecord{{Identifier: power_configuration.BatterySource2Voltage, Status: 1}, {Identifier: power_configuration.BatterySource2PercentageRemaining, Status: 1}}, nil)

		mzc.On("ReadAttributes", mock.Anything, ieeeAddress, true, zcl.PowerConfigurationId, zigbee.NoManufacturer, zigbee.Endpoint(2), zigbee.Endpoint(1), uint8(4), []zcl.AttributeID{
			power_configuration.BatterySource3Voltage,
			power_configuration.BatterySource3PercentageRemaining,
		}).Return([]global.ReadAttributeResponseRecord{{Identifier: power_configuration.BatterySource3Voltage, Status: 1}, {Identifier: power_configuration.BatterySource3PercentageRemaining, Status: 1}}, nil)

		attached, err := i.Enumerate(context.Background(), map[string]any{"ZigbeeEndpoint": zigbee.Endpoint(1), "ZigbeePowerConfigurationClusterPresent": true})
		assert.NoError(t, err)
		assert.True(t, attached)

		assert.True(t, i.mainsPresent)
		assert.True(t, i.mainsVoltagePresent)
		assert.True(t, i.mainsFrequencyPresent)
		assert.True(t, i.batteryPresent[0])
		assert.True(t, i.batteryVoltagePresent[0])
		assert.True(t, i.batteryPercentagePresent[0])

		v, _ := i.s.Bool(MainsPresentKey)
		assert.True(t, v)
		v, _ = i.s.Bool(MainsVoltagePresentKey)
		assert.True(t, v)
		v, _ = i.s.Bool(MainsFrequencyPresentKey)
		assert.True(t, v)
		v, _ = i.s.Bool(BatteryPresent(0))
		assert.True(t, v)
		v, _ = i.s.Bool(BatteryVoltagePresent(0))
		assert.True(t, v)
		v, _ = i.s.Bool(BatteryPercentagePresent(0))
		assert.True(t, v)
	})
}

func TestImplementation_Detach(t *testing.T) {
	t.Run("detached attribute monitors on detach", func(t *testing.T) {
		mm := &attribute.MockMonitor{}
		defer mm.AssertExpectations(t)

		mm.On("Detach", mock.Anything, true).Return(nil).Times(8)

		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsVoltagePresent = true
		i.mainsFrequencyPresent = true
		i.batteryPercentagePresent[0] = true
		i.batteryVoltagePresent[0] = true
		i.batteryPercentagePresent[1] = true
		i.batteryVoltagePresent[1] = true
		i.batteryPercentagePresent[2] = true
		i.batteryVoltagePresent[2] = true

		i.mainsVoltageMonitor = mm
		i.mainsFrequencyMonitor = mm
		i.batteryPercentageMonitor[0] = mm
		i.batteryVoltageMonitor[0] = mm
		i.batteryPercentageMonitor[1] = mm
		i.batteryVoltageMonitor[1] = mm
		i.batteryPercentageMonitor[2] = mm
		i.batteryVoltageMonitor[2] = mm

		err := i.Detach(context.TODO(), implcaps.NoLongerEnumerated)
		assert.NoError(t, err)
	})
}

func TestImplementation_update(t *testing.T) {
	t.Run("mains voltage updates the state correctly, sending event if change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.PowerStatusUpdate)
			assert.True(t, ok)
			assert.InEpsilon(t, 239.5, e.PowerStatus.Mains[0].Voltage, 0.001)
		})

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsPresent = true
		i.mainsVoltagePresent = true

		i.s.Set(MainsVoltageKey, 239.0)

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(power_configuration.MainsVoltage, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(2395),
		})

		state, _ := i.Status(context.TODO())
		assert.InEpsilon(t, 239.5, state.Mains[0].Voltage, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("mains frequency updates the state correctly, sending event if change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.PowerStatusUpdate)
			assert.True(t, ok)
			assert.InEpsilon(t, 50.0, e.PowerStatus.Mains[0].Frequency, 0.001)
		})

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsPresent = true
		i.mainsFrequencyPresent = true

		i.s.Set(MainsFrequencyKey, 48)

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(power_configuration.MainsFrequency, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint64(100),
		})

		state, _ := i.Status(context.TODO())
		assert.InEpsilon(t, 50, state.Mains[0].Frequency, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("battery voltage updates the state correctly, sending event if change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.PowerStatusUpdate)
			assert.True(t, ok)
			assert.InEpsilon(t, 3.0, e.PowerStatus.Battery[0].Voltage, 0.001)
		})

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.batteryPresent[0] = true
		i.batteryVoltagePresent[0] = true

		i.s.Set(BatteryVoltage(0), 3.1)

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(power_configuration.BatteryVoltage, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint64(30),
		})

		state, _ := i.Status(context.TODO())
		assert.InEpsilon(t, 3.0, state.Battery[0].Voltage, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("battery percent updates the state correctly, sending event if change", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("SendEvent", mock.Anything).Run(func(args mock.Arguments) {
			e, ok := args.Get(0).(capabilities.PowerStatusUpdate)
			assert.True(t, ok)
			assert.InEpsilon(t, 0.995, e.PowerStatus.Battery[0].Remaining, 0.001)
		})

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.batteryPresent[0] = true
		i.batteryPercentagePresent[0] = true

		i.s.Set(BatteryPercentage(1), 100)

		lastUpdated := time.Now().Add(-5 * time.Minute)
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(power_configuration.BatteryPercentageRemaining, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt8,
			Value:    uint64(199),
		})

		state, _ := i.Status(context.TODO())
		assert.InEpsilon(t, 0.995, state.Battery[0].Remaining, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Greater(t, lct, lastUpdated)
	})

	t.Run("updates the state correctly, no event", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsPresent = true
		i.mainsVoltagePresent = true

		i.s.Set(MainsVoltageKey, 239.0)

		lastUpdated := time.UnixMilli(time.Now().Add(-5 * time.Minute).UnixMilli())
		converter.Store(i.s, implcaps.LastUpdatedKey, lastUpdated, converter.TimeEncoder)
		converter.Store(i.s, implcaps.LastChangedKey, lastUpdated, converter.TimeEncoder)

		i.update(power_configuration.MainsVoltage, zcl.AttributeDataTypeValue{
			DataType: zcl.TypeUnsignedInt16,
			Value:    uint64(2390),
		})

		state, _ := i.Status(context.TODO())
		assert.InEpsilon(t, 239.0, state.Mains[0].Voltage, 0.001)

		lut, _ := i.LastUpdateTime(context.TODO())
		assert.Greater(t, lut, lastUpdated)

		lct, _ := i.LastChangeTime(context.TODO())
		assert.Equal(t, lct, lastUpdated)
	})
}

func TestImplementation_Reading(t *testing.T) {
	t.Run("returns the current power status", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
		i.s = memory.New()

		i.mainsPresent = true
		i.mainsVoltagePresent = true
		i.mainsFrequencyPresent = true
		i.batteryPresent[0] = true
		i.batteryPresent[1] = true
		i.batteryPresent[2] = true
		i.batteryPercentagePresent[0] = true
		i.batteryPercentagePresent[1] = true
		i.batteryPercentagePresent[2] = true
		i.batteryVoltagePresent[0] = true
		i.batteryVoltagePresent[1] = true
		i.batteryVoltagePresent[2] = true

		i.s.Set(MainsVoltageKey, 230.0)
		i.s.Set(MainsFrequencyKey, 50.0)
		i.s.Set(BatteryPercentage(0), 100.0)
		i.s.Set(BatteryPercentage(1), 75.0)
		i.s.Set(BatteryPercentage(2), 50.0)
		i.s.Set(BatteryVoltage(0), 3.0)
		i.s.Set(BatteryVoltage(1), 2.9)
		i.s.Set(BatteryVoltage(2), 2.8)

		status, err := i.Status(context.TODO())
		assert.NoError(t, err)

		assert.Len(t, status.Mains, 1)
		assert.Len(t, status.Battery, 3)

		assert.Equal(t, capabilities.Voltage|capabilities.Frequency, status.Mains[0].Present)
		assert.Equal(t, capabilities.Voltage|capabilities.Remaining, status.Battery[0].Present)
		assert.Equal(t, capabilities.Voltage|capabilities.Remaining, status.Battery[1].Present)
		assert.Equal(t, capabilities.Voltage|capabilities.Remaining, status.Battery[2].Present)

		assert.InDelta(t, 230.0, status.Mains[0].Voltage, 0.0001)
		assert.InDelta(t, 50.0, status.Mains[0].Frequency, 0.0001)

		assert.InDelta(t, 3.0, status.Battery[0].Voltage, 0.0001)
		assert.InDelta(t, 1.0, status.Battery[0].Remaining, 0.0001)

		assert.InDelta(t, 2.9, status.Battery[1].Voltage, 0.0001)
		assert.InDelta(t, 0.75, status.Battery[1].Remaining, 0.0001)

		assert.InDelta(t, 2.8, status.Battery[2].Voltage, 0.0001)
		assert.InDelta(t, 0.5, status.Battery[2].Remaining, 0.0001)
	})
}

func TestImplementation_LastTimes(t *testing.T) {
	t.Run("returns the last updated and changed times", func(t *testing.T) {
		mzi := &implcaps.MockZDAInterface{}
		defer mzi.AssertExpectations(t)

		mzi.On("Logger").Return(logwrap.New(discard.Discard()))

		i := NewPowerSupply(mzi)
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
