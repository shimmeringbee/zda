package power_suply

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/persistence/converter"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"math"
	"time"
)

var _ capabilities.PowerSupply = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)
var _ implcaps.ZDACapability = (*Implementation)(nil)

const MainsPresentKey = "MainsPresent"

const MainsVoltageKey = "MainsVoltage"
const MainsFrequencyKey = "MainsFrequency"

var BatteryVoltage = func(n int) string { return fmt.Sprintf("BatteryVoltage%d", n) }
var BatteryPercentage = func(n int) string { return fmt.Sprintf("BatteryPercentage%d", n) }

const MainsVoltagePresentKey = "MainsVoltagePresent"
const MainsFrequencyPresentKey = "MainsFrequencyPresent"

var BatteryPresent = func(n int) string { return fmt.Sprintf("BatteryPresent%d", n) }

var BatteryVoltagePresent = func(n int) string { return fmt.Sprintf("BatteryVoltage%dPresent", n) }
var BatteryPercentagePresent = func(n int) string { return fmt.Sprintf("BatteryPercentage%dPresent", n) }

func NewPowerSupply(zi implcaps.ZDAInterface) *Implementation {
	return &Implementation{zi: zi, l: zi.Logger()}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	zi implcaps.ZDAInterface
	l  logwrap.Logger

	remoteEndpoint zigbee.Endpoint

	mainsVoltageMonitor      attribute.Monitor
	mainsFrequencyMonitor    attribute.Monitor
	batteryVoltageMonitor    [3]attribute.Monitor
	batteryPercentageMonitor [3]attribute.Monitor

	mainsPresent             bool
	mainsVoltagePresent      bool
	mainsFrequencyPresent    bool
	batteryPresent           [3]bool
	batteryVoltagePresent    [3]bool
	batteryPercentagePresent [3]bool
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.PowerSupplyFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.PowerSupplyFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s

	i.mainsVoltageMonitor = i.zi.NewAttributeMonitor()
	i.mainsVoltageMonitor.Init(s.Section("AttributeMonitor", "MainsVoltage"), d, i.update)
	i.mainsFrequencyMonitor = i.zi.NewAttributeMonitor()
	i.mainsFrequencyMonitor.Init(s.Section("AttributeMonitor", "MainsFrequency"), d, i.update)

	for n := range len(i.batteryVoltageMonitor) {
		i.batteryVoltageMonitor[n] = i.zi.NewAttributeMonitor()
		i.batteryVoltageMonitor[n].Init(s.Section("AttributeMonitor", fmt.Sprintf("BatteryVoltage%d", n)), d, i.update)
		i.batteryPercentageMonitor[n] = i.zi.NewAttributeMonitor()
		i.batteryPercentageMonitor[n].Init(s.Section("AttributeMonitor", fmt.Sprintf("BatteryPercentage%d", n)), d, i.update)
	}
}

func (i *Implementation) Load(ctx context.Context) (bool, error) {
	i.mainsPresent, _ = i.s.Bool(MainsPresentKey)
	i.mainsVoltagePresent, _ = i.s.Bool(MainsVoltagePresentKey)
	i.mainsFrequencyPresent, _ = i.s.Bool(MainsFrequencyPresentKey)

	if i.mainsVoltagePresent {
		if err := i.mainsVoltageMonitor.Load(ctx); err != nil {
			i.l.Warn(ctx, "Failed to attach mains voltage monitor.", logwrap.Err(err))
			return false, fmt.Errorf("mains voltage monitor, attach failed: %w", err)
		}
	}

	if i.mainsFrequencyPresent {
		if err := i.mainsFrequencyMonitor.Load(ctx); err != nil {
			i.l.Warn(ctx, "Failed to attach mains frequency monitor.", logwrap.Err(err))
			return false, fmt.Errorf("mains voltage frequency, attach failed: %w", err)
		}
	}

	for n := range 3 {
		i.batteryPresent[n], _ = i.s.Bool(BatteryPresent(n))
		i.batteryPercentagePresent[n], _ = i.s.Bool(BatteryPercentagePresent(n))
		i.batteryVoltagePresent[n], _ = i.s.Bool(BatteryVoltagePresent(n))

		if i.batteryPercentagePresent[n] {
			if err := i.batteryPercentageMonitor[n].Load(ctx); err != nil {
				i.l.Warn(ctx, "Failed to attach battery percentage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
				return false, fmt.Errorf("battery %d percentage monitor, attach failed: %w", n, err)
			}
		}

		if i.batteryVoltagePresent[n] {
			if err := i.batteryVoltageMonitor[n].Load(ctx); err != nil {
				i.l.Warn(ctx, "Failed to attach battery voltage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
				return false, fmt.Errorf("battery %d voltage monitor, attach failed: %w", n, err)
			}
		}
	}

	return true, nil
}

func (i *Implementation) enumerateBasicCluster(pctx context.Context) (bool, error) {
	var lastError error
	attach := false

	ctx, done := context.WithTimeout(pctx, 1*time.Second)
	defer done()

	ieee, localEndpoint, ack, seq := i.zi.TransmissionLookup(i.d, zigbee.ProfileHomeAutomation)
	i.l.Info(ctx, "Reading basic power configuration data.")
	if resp, err := i.zi.ZCLCommunicator().ReadAttributes(ctx, ieee, ack, zcl.BasicId, zigbee.NoManufacturer, localEndpoint, i.remoteEndpoint, seq, []zcl.AttributeID{basic.PowerSource}); err != nil {
		lastError = err
		i.l.Warn(ctx, "Failed to read basic power configuration.", logwrap.Err(err))
	} else if len(resp) == 0 || resp[0].Status != 0 || resp[0].Identifier != basic.PowerSource {
		i.l.Warn(ctx, "Device did not respond to read attribute for mandatory PowerSource attribute!")
	} else {
		val64, ok64 := resp[0].DataTypeValue.Value.(uint64)
		val8, ok8 := resp[0].DataTypeValue.Value.(uint8)

		val := val64 + uint64(val8)

		if ok8 || ok64 {
			attach = true

			switch val & 0x0f {
			case 0x01, 0x02, 0x04, 0x05, 0x06:
				i.mainsPresent = true
			case 0x03:
				i.batteryPresent[0] = true
			}

			if val&0x80 == 0x80 {
				if i.batteryPresent[0] == true {
					i.batteryPresent[1] = true
				} else {
					i.batteryPresent[0] = true
				}
			}
		} else {
			i.l.Warn(ctx, "Device did not return int coercible value.", logwrap.Datum("DataType", resp[0].DataTypeValue.DataType))
		}
	}

	if i.mainsPresent {
		i.s.Set(MainsPresentKey, true)
	}

	for n := range 2 {
		if i.batteryPresent[n] {
			i.s.Set(BatteryPresent(n), true)
		}
	}

	return attach, lastError
}

func (i *Implementation) enumeratePowerConfigurationCluster(pctx context.Context) (bool, error) {
	var lastError error
	attach := false

	ieee, localEndpoint, ack, seq := i.zi.TransmissionLookup(i.d, zigbee.ProfileHomeAutomation)
	ctx, done := context.WithTimeout(pctx, 5*time.Second)
	i.l.Info(ctx, "Reading PowerConfiguration cluster for mains data.")
	if resp, err := i.zi.ZCLCommunicator().ReadAttributes(ctx, ieee, ack, zcl.PowerConfigurationId, zigbee.NoManufacturer, localEndpoint, i.remoteEndpoint, seq, []zcl.AttributeID{
		power_configuration.MainsVoltage,
		power_configuration.MainsFrequency,
	}); err != nil {
		lastError = err
		i.l.Warn(ctx, "Errored reading mains from power configuration.", logwrap.Err(err))
	} else {
		for _, d := range resp {
			if d.Status == 0 {
				attach = true

				switch d.Identifier {
				case power_configuration.MainsVoltage:
					i.mainsPresent = true
					i.mainsVoltagePresent = true
				case power_configuration.MainsFrequency:
					i.mainsPresent = true
					i.mainsFrequencyPresent = true
				}
			}
		}
	}
	done()

	batteryPercentageAttributes := []zcl.AttributeID{power_configuration.BatteryPercentageRemaining, power_configuration.BatterySource2PercentageRemaining, power_configuration.BatterySource3PercentageRemaining}
	batteryVoltageAttributes := []zcl.AttributeID{power_configuration.BatteryVoltage, power_configuration.BatterySource2Voltage, power_configuration.BatterySource3Voltage}

	for n := range 3 {
		_, _, _, seq = i.zi.TransmissionLookup(i.d, zigbee.ProfileHomeAutomation)
		ctx, done := context.WithTimeout(pctx, 5*time.Second)
		i.l.Info(ctx, "Reading PowerConfiguration cluster for battery data.", logwrap.Datum("Battery", n))
		if resp, err := i.zi.ZCLCommunicator().ReadAttributes(ctx, ieee, ack, zcl.PowerConfigurationId, zigbee.NoManufacturer, localEndpoint, i.remoteEndpoint, seq, []zcl.AttributeID{
			batteryVoltageAttributes[n],
			batteryPercentageAttributes[n],
		}); err != nil {
			lastError = err
			i.l.LogWarn(ctx, "Failed to query battery status.", logwrap.Err(err), logwrap.Datum("Battery", n))
		} else {
			for _, d := range resp {
				attach = true

				if d.Status == 0 {
					switch d.Identifier {
					case batteryPercentageAttributes[n]:
						i.batteryPresent[n] = true
						i.batteryPercentagePresent[n] = true
					case batteryVoltageAttributes[n]:
						i.batteryPresent[n] = true
						i.batteryVoltagePresent[n] = true
					}
				}
			}
		}
		done()
	}

	i.s.Set(MainsPresentKey, i.mainsPresent)

	if i.mainsVoltagePresent {
		i.s.Set(MainsVoltagePresentKey, true)

		if err := i.mainsVoltageMonitor.Attach(pctx, i.remoteEndpoint, zcl.PowerConfigurationId, power_configuration.MainsVoltage, zcl.TypeUnsignedInt16, attribute.ReportingConfig{Mode: attribute.AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: uint(5)}, attribute.PollingConfig{Mode: attribute.PollIfReportingFailed, Interval: 5 * time.Minute}); err != nil {
			lastError = err
			i.l.Warn(pctx, "Errored attaching mains voltage monitor.", logwrap.Err(err))
		}
	}

	if i.mainsFrequencyPresent {
		i.s.Set(MainsFrequencyPresentKey, true)

		if err := i.mainsVoltageMonitor.Attach(pctx, i.remoteEndpoint, zcl.PowerConfigurationId, power_configuration.MainsFrequency, zcl.TypeUnsignedInt8, attribute.ReportingConfig{Mode: attribute.AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: uint(1)}, attribute.PollingConfig{Mode: attribute.PollIfReportingFailed, Interval: 5 * time.Minute}); err != nil {
			lastError = err
			i.l.Warn(pctx, "Errored attaching mains frequency monitor.", logwrap.Err(err))
		}
	}

	for n := range 3 {
		i.s.Set(BatteryPresent(n), i.batteryPresent[n])

		if i.batteryPercentagePresent[n] {
			i.s.Set(BatteryPercentagePresent(n), true)

			if err := i.batteryPercentageMonitor[n].Attach(pctx, i.remoteEndpoint, zcl.PowerConfigurationId, batteryPercentageAttributes[n], zcl.TypeUnsignedInt8, attribute.ReportingConfig{Mode: attribute.AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: uint(1)}, attribute.PollingConfig{Mode: attribute.PollIfReportingFailed, Interval: 5 * time.Minute}); err != nil {
				lastError = err
				i.l.Warn(pctx, "Errored attaching battery percentage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
			}
		}

		if i.batteryVoltagePresent[n] {
			i.s.Set(BatteryVoltagePresent(n), true)

			if err := i.batteryVoltageMonitor[n].Attach(pctx, i.remoteEndpoint, zcl.PowerConfigurationId, batteryVoltageAttributes[n], zcl.TypeUnsignedInt8, attribute.ReportingConfig{Mode: attribute.AttemptConfigureReporting, MinimumInterval: 1 * time.Minute, MaximumInterval: 5 * time.Minute, ReportableChange: uint(1)}, attribute.PollingConfig{Mode: attribute.PollIfReportingFailed, Interval: 5 * time.Minute}); err != nil {
				lastError = err
				i.l.Warn(pctx, "Errored attaching battery percentage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
			}
		}
	}

	return attach, lastError
}

func (i *Implementation) Enumerate(ctx context.Context, m map[string]any) (bool, error) {
	var lastError error
	attach := false

	i.remoteEndpoint = implcaps.Get(m, "ZigbeeEndpoint", zigbee.Endpoint(1))
	i.s.Set(implcaps.RemoteEndpointKey, int(i.remoteEndpoint))

	if implcaps.Get(m, "ZigbeeBasicClusterPresent", false) {
		if bcAttach, err := i.enumerateBasicCluster(ctx); err != nil {
			lastError = fmt.Errorf("enumerating basic cluster: %w", err)
		} else if bcAttach {
			attach = true
		}
	}

	if implcaps.Get(m, "ZigbeePowerConfigurationClusterPresent", false) {
		if pcAttach, err := i.enumeratePowerConfigurationCluster(ctx); err != nil {
			lastError = fmt.Errorf("enumerating power configuration cluster: %w", err)
		} else if pcAttach {
			attach = true
		}
	}

	return attach, lastError
}

func (i *Implementation) Detach(ctx context.Context, detachType implcaps.DetachType) error {
	var lastError error

	if i.mainsVoltagePresent {
		if err := i.mainsVoltageMonitor.Detach(ctx, detachType == implcaps.NoLongerEnumerated); err != nil {
			lastError = err
			i.l.Warn(ctx, "Failed to attach mains voltage monitor.", logwrap.Err(err))
		}
	}

	if i.mainsFrequencyPresent {
		if err := i.mainsFrequencyMonitor.Detach(ctx, detachType == implcaps.NoLongerEnumerated); err != nil {
			lastError = err
			i.l.Warn(ctx, "Failed to attach mains frequency monitor.", logwrap.Err(err))
		}
	}

	for n := range 3 {
		if i.batteryPercentagePresent[n] {
			if err := i.batteryPercentageMonitor[n].Detach(ctx, detachType == implcaps.NoLongerEnumerated); err != nil {
				lastError = err
				i.l.Warn(ctx, "Failed to attach battery percentage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
			}
		}

		if i.batteryVoltagePresent[n] {
			if err := i.batteryVoltageMonitor[n].Detach(ctx, detachType == implcaps.NoLongerEnumerated); err != nil {
				lastError = err
				i.l.Warn(ctx, "Failed to attach battery voltage monitor.", logwrap.Err(err), logwrap.Datum("Battery", n))
			}
		}
	}

	return lastError
}

func (i *Implementation) ImplName() string {
	return "ZCLPowerSupply"
}

func (i *Implementation) LastUpdateTime(_ context.Context) (time.Time, error) {
	t, _ := converter.Retrieve(i.s, implcaps.LastUpdatedKey, converter.TimeDecoder)
	return t, nil
}

func (i *Implementation) LastChangeTime(_ context.Context) (time.Time, error) {
	t, _ := converter.Retrieve(i.s, implcaps.LastChangedKey, converter.TimeDecoder)
	return t, nil
}

func (i *Implementation) Status(_ context.Context) (capabilities.PowerState, error) {
	ret := &capabilities.PowerState{
		Mains:   nil,
		Battery: nil,
	}

	if i.mainsPresent {
		var present capabilities.PowerStatusPresent
		voltage := 0.0
		frequency := 0.0

		if i.mainsVoltagePresent {
			present |= capabilities.Voltage
			voltage, _ = i.s.Float(MainsVoltageKey)
		}

		if i.mainsFrequencyPresent {
			present |= capabilities.Frequency
			frequency, _ = i.s.Float(MainsFrequencyKey)
		}

		ret.Mains = append(ret.Mains, capabilities.PowerMainsState{
			Voltage:   voltage,
			Frequency: frequency,
			Available: false,
			Present:   present,
		})
	}

	for n := range 3 {
		if i.batteryPresent[n] {
			var present capabilities.PowerStatusPresent
			remaining := 0.0
			voltage := 0.0

			if i.batteryPercentagePresent[n] {
				present |= capabilities.Remaining
				remaining, _ = i.s.Float(BatteryPercentage(n))
				remaining /= 100
			}

			if i.batteryVoltagePresent[n] {
				present |= capabilities.Voltage
				voltage, _ = i.s.Float(BatteryVoltage(n))
			}

			ret.Battery = append(ret.Battery, capabilities.PowerBatteryState{
				Voltage:   voltage,
				Remaining: remaining,
				Available: false,
				Present:   present,
			})
		}
	}

	return *ret, nil
}

func (i *Implementation) update(id zcl.AttributeID, value zcl.AttributeDataTypeValue) {
	announce := false

	if raw, ok := value.Value.(uint64); ok {
		switch id {
		case power_configuration.MainsVoltage:
			newVoltage := float64(raw) / 10.0
			currentVoltage, _ := i.s.Float(MainsVoltageKey)

			announce = math.Abs(newVoltage-currentVoltage) >= 0.1
			i.s.Set(MainsVoltageKey, newVoltage)

		case power_configuration.MainsFrequency:
			newFrequency := float64(raw) / 2.0
			currentFrequency, _ := i.s.Float(MainsFrequencyKey)

			announce = math.Abs(newFrequency-currentFrequency) >= 1
			i.s.Set(MainsFrequencyKey, newFrequency)

		case power_configuration.BatteryVoltage, power_configuration.BatterySource2Voltage, power_configuration.BatterySource3Voltage:
			battery := attributeIdToBattery(id)
			newVoltage := float64(raw) / 10.0
			currentVoltage, _ := i.s.Float(BatteryVoltage(battery))

			announce = math.Abs(newVoltage-currentVoltage) >= 0.05
			i.s.Set(BatteryVoltage(battery), newVoltage)

		case power_configuration.BatteryPercentageRemaining, power_configuration.BatterySource2PercentageRemaining, power_configuration.BatterySource3PercentageRemaining:
			battery := attributeIdToBattery(id)
			newPercentage := float64(raw) / 2
			currentPercentage, _ := i.s.Float(BatteryPercentage(battery))

			announce = math.Abs(newPercentage-currentPercentage) >= 0.1
			i.s.Set(BatteryPercentage(battery), newPercentage)
		}
	}

	if announce {
		converter.Store(i.s, implcaps.LastChangedKey, time.Now(), converter.TimeEncoder)
		s, _ := i.Status(context.Background())
		i.zi.SendEvent(capabilities.PowerStatusUpdate{Device: i.d, PowerStatus: s})
	}

	converter.Store(i.s, implcaps.LastUpdatedKey, time.Now(), converter.TimeEncoder)
}

func attributeIdToBattery(id zcl.AttributeID) int {
	if id&0x0020 == 0x0020 {
		return 0
	} else if id&0x0040 == 0x0040 {
		return 1
	} else if id&0x0060 == 0x0060 {
		return 2
	}

	return 0
}
