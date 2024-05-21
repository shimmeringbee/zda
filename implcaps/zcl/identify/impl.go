package identify

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/identify"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"math"
	"time"
)

var _ capabilities.Identify = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)
var _ implcaps.ZDACapability = (*Implementation)(nil)

const RemainingDurationKey = "RemainingDuration"

func NewIdentify(zi implcaps.ZDAInterface) *Implementation {
	zi.ZCLRegister(identify.Register)
	return &Implementation{zi: zi}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	am attribute.Monitor
	zi implcaps.ZDAInterface

	clusterId      zigbee.ClusterID
	remoteEndpoint zigbee.Endpoint
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.IdentifyFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.IdentifyFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s

	i.am = i.zi.NewAttributeMonitor()
	i.am.Init(s.Section("AttributeMonitor", "IdentifyTime"), d, i.update)
}

func (i *Implementation) Load(ctx context.Context) (bool, error) {
	if v, ok := i.s.Int(implcaps.RemoteEndpointKey); ok {
		i.remoteEndpoint = zigbee.Endpoint(v)
	} else {
		//	i.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", implcaps.RemoteEndpointKey))
		return false, fmt.Errorf("monitor missing config parameter: %s", implcaps.RemoteEndpointKey)
	}

	if v, ok := i.s.Int(implcaps.ClusterIdKey); ok {
		i.clusterId = zigbee.ClusterID(v)
	} else {
		//		i.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", implcaps.ClusterIdKey))
		return false, fmt.Errorf("monitor missing config parameter: %s", implcaps.ClusterIdKey)
	}

	if err := i.am.Load(ctx); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (i *Implementation) Enumerate(ctx context.Context, m map[string]any) (bool, error) {
	i.remoteEndpoint = implcaps.Get(m, "ZigbeeEndpoint", zigbee.Endpoint(1))
	i.clusterId = implcaps.Get(m, "ZigbeeIdentifyClusterID", zcl.IdentifyId)
	attributeId := implcaps.Get(m, "ZigbeeIdentifyDurationAttributeID", identify.IdentifyTime)

	i.s.Set(implcaps.ClusterIdKey, int(i.clusterId))
	i.s.Set(implcaps.RemoteEndpointKey, int(i.remoteEndpoint))

	reporting := attribute.ReportingConfig{
		Mode:             attribute.AttemptConfigureReporting,
		MinimumInterval:  1 * time.Second,
		MaximumInterval:  5 * time.Minute,
		ReportableChange: uint(1),
	}

	polling := attribute.PollingConfig{
		Mode:     attribute.PollIfReportingFailed,
		Interval: 1 * time.Minute,
	}

	if err := i.am.Attach(ctx, i.remoteEndpoint, i.clusterId, attributeId, zcl.TypeUnsignedInt16, reporting, polling); err != nil {
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
	return "ZCLIdentify"
}

func (i *Implementation) update(_ zcl.AttributeID, v zcl.AttributeDataTypeValue) {
	if v.DataType == zcl.TypeUnsignedInt16 {
		if seconds, ok := v.Value.(uint64); ok {
			i.updateDuration(time.Duration(seconds) * time.Second)
		}
	}
}

func (i *Implementation) updateDuration(duration time.Duration) {
	newEndTime := time.Now()

	if duration > 0 {
		newEndTime = newEndTime.Add(duration)
	}

	currentEndTimeInMs, _ := i.s.Int(RemainingDurationKey, int(time.Now().UnixMilli()))
	currentEndTime := time.UnixMilli(int64(currentEndTimeInMs))

	diffDuration := newEndTime.Sub(currentEndTime)

	if diffDuration >= (250*time.Millisecond) || (currentEndTime != newEndTime && diffDuration < 0) {
		i.s.Set(RemainingDurationKey, newEndTime.UnixMilli())
		i.s.Set(implcaps.LastChangedKey, time.Now().UnixMilli())

		if diffDuration < 0 {
			diffDuration = 0
		}

		i.zi.SendEvent(capabilities.IdentifyUpdate{Device: i.d, State: capabilities.IdentifyState{
			Identifying: diffDuration > 0,
			Remaining:   diffDuration,
		}})
	}

	i.s.Set(implcaps.LastUpdatedKey, time.Now().UnixMilli())
}

func (i *Implementation) LastUpdateTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int(implcaps.LastUpdatedKey)
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) LastChangeTime(_ context.Context) (time.Time, error) {
	t, _ := i.s.Int(implcaps.LastChangedKey)
	return time.UnixMilli(int64(t)), nil
}

func (i *Implementation) Status(_ context.Context) (capabilities.IdentifyState, error) {
	endTimeInMs, _ := i.s.Int(RemainingDurationKey)
	endTime := time.UnixMilli(int64(endTimeInMs))

	diffDuration := endTime.Sub(time.Now())

	if diffDuration < 0 {
		diffDuration = 0
	}

	return capabilities.IdentifyState{
		Identifying: diffDuration > 0,
		Remaining:   diffDuration,
	}, nil
}

func (i *Implementation) Identify(ctx context.Context, duration time.Duration) error {
	ieee, localEndpoint, ack, seq := i.zi.TransmissionLookup(i.d, zigbee.ProfileHomeAutomation)

	identifySeconds := float64(duration / time.Second)
	identifyTime := uint16(math.Min(0xffff, identifySeconds))

	if err := i.zi.ZCLCommunicator().Request(ctx, ieee, ack, zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: seq,
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           i.clusterId,
		SourceEndpoint:      localEndpoint,
		DestinationEndpoint: i.remoteEndpoint,
		CommandIdentifier:   identify.IdentifyId,
		Command:             &identify.Identify{IdentifyTime: identifyTime},
	}); err != nil {
		return err
	}

	i.updateDuration(time.Duration(identifyTime) * time.Second)

	return nil
}
