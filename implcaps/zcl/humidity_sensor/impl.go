package humidity_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/relative_humidity_measurement"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"math"
	"time"
)

var _ capabilities.RelativeHumiditySensor = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)
var _ implcaps.ZDACapability = (*Implementation)(nil)

func NewHumiditySensor(zi implcaps.ZDAInterface) *Implementation {
	return &Implementation{zi: zi}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	am attribute.Monitor
	zi implcaps.ZDAInterface
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.RelativeHumiditySensorFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.RelativeHumiditySensorFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s

	i.am = i.zi.NewAttributeMonitor()
	i.am.Init(s.Section("AttributeMonitor", "HumidityReading"), d, i.update)
}

func (i *Implementation) Load(ctx context.Context) (bool, error) {
	if err := i.am.Load(ctx); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (i *Implementation) Enumerate(ctx context.Context, m map[string]any) (bool, error) {
	endpoint := implcaps.Get(m, "ZigbeeEndpoint", zigbee.Endpoint(1))
	clusterId := implcaps.Get(m, "ZigbeeHumiditySensorClusterID", zcl.RelativeHumidityMeasurementId)
	attributeId := implcaps.Get(m, "ZigbeeHumiditySensorAttributeID", relative_humidity_measurement.MeasuredValue)

	reporting := attribute.ReportingConfig{
		Mode:             attribute.AttemptConfigureReporting,
		MinimumInterval:  1 * time.Minute,
		MaximumInterval:  5 * time.Minute,
		ReportableChange: uint(100),
	}

	polling := attribute.PollingConfig{
		Mode:     attribute.PollIfReportingFailed,
		Interval: 1 * time.Minute,
	}

	if err := i.am.Attach(ctx, endpoint, clusterId, attributeId, zcl.TypeUnsignedInt16, reporting, polling); err != nil {
		return false, err
	}

	return true, nil
}

func (i *Implementation) Detach(ctx context.Context, detachType implcaps.DetachType) error {
	if err := i.am.Detach(ctx, detachType == implcaps.NoLongerEnumerated); err != nil {
		return err
	}

	return nil
}

func (i *Implementation) ImplName() string {
	return "ZCLHumiditySensor"
}

func (i *Implementation) update(_ zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeUnsignedInt16 {
		if value, ok := v.Value.(uint64); ok {
			newRatio := float64(value) / 10000.0
			currentRatio, _ := i.s.Float("Reading")

			if math.Abs(newRatio-currentRatio) > 0.01 {
				i.s.Set("Reading", newRatio)
				i.s.Set("LastChanged", time.Now().UnixMilli())

				i.zi.SendEvent(capabilities.RelativeHumiditySensorState{Device: i.d, State: []capabilities.RelativeHumidityReading{{Value: newRatio}}})
			}

			i.s.Set("LastUpdated", time.Now().UnixMilli())
		}
	}
}

func (i *Implementation) LastUpdateTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int("LastUpdated")
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) LastChangeTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int("LastChanged")
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) Reading(_ context.Context) ([]capabilities.RelativeHumidityReading, error) {
	k, _ := i.s.Float("Reading")

	return []capabilities.RelativeHumidityReading{
		{
			Value: k,
		},
	}, nil
}
