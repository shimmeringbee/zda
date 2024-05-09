package pressure_sensor

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/pressure_measurement"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"math"
	"time"
)

var _ capabilities.PressureSensor = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)
var _ implcaps.ZDACapability = (*Implementation)(nil)

func NewPressureSensor(zi implcaps.ZDAInterface) *Implementation {
	return &Implementation{zi: zi}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	am attribute.Monitor
	zi implcaps.ZDAInterface
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.PressureSensorFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.PressureSensorFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s

	i.am = i.zi.NewAttributeMonitor()
	i.am.Init(s.Section("AttributeMonitor", "PressureReading"), d, i.update)
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
	clusterId := implcaps.Get(m, "ZigbeePressureMeasurementClusterID", zcl.PressureMeasurementId)
	attributeId := implcaps.Get(m, "ZigbeePressureMeasurementAttributeID", pressure_measurement.MeasuredValue)

	reporting := attribute.ReportingConfig{
		Mode:             attribute.AttemptConfigureReporting,
		MinimumInterval:  1 * time.Minute,
		MaximumInterval:  5 * time.Minute,
		ReportableChange: 10,
	}

	polling := attribute.PollingConfig{
		Mode:     attribute.PollIfReportingFailed,
		Interval: 1 * time.Minute,
	}

	if err := i.am.Attach(ctx, endpoint, clusterId, attributeId, zcl.TypeSignedInt16, reporting, polling); err != nil {
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
	return "ZCLPressureSensor"
}

func (i *Implementation) update(_ zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeSignedInt16 {
		if value, ok := v.Value.(int64); ok {
			newPressure := float64(value) / 10.0
			currentPressure, _ := i.s.Float(implcaps.ReadingKey)

			if math.Abs(newPressure-currentPressure) > 0.1 {
				i.s.Set(implcaps.ReadingKey, newPressure)
				i.s.Set(implcaps.LastChangedKey, time.Now().UnixMilli())

				i.zi.SendEvent(capabilities.PressureSensorState{Device: i.d, State: []capabilities.PressureReading{{Value: newPressure}}})
			}

			i.s.Set(implcaps.LastUpdatedKey, time.Now().UnixMilli())
		}
	}
}

func (i *Implementation) LastUpdateTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int(implcaps.LastUpdatedKey)
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) LastChangeTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int(implcaps.LastChangedKey)
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) Reading(_ context.Context) ([]capabilities.PressureReading, error) {
	k, _ := i.s.Float(implcaps.ReadingKey)

	return []capabilities.PressureReading{
		{
			Value: k,
		},
	}, nil
}
