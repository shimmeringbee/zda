package level

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/level"
	"math"
	"time"
)

var _ capabilities.Level = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)

const PollAfterSetDelay = 100 * time.Millisecond

func (i *Implementation) Change(ctx context.Context, device da.Device, f float64, duration time.Duration) error {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.LevelFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	tenthsOfSecond := uint16(math.Round(float64(duration / (100 * time.Millisecond))))
	changeLevel := uint8(math.Round(f * 254.0))

	err := i.supervisor.ZCL().SendCommand(ctx, d, i.data[d.Identifier].Endpoint, zcl.LevelControlId, &level.MoveToLevelWithOnOff{
		Level:          changeLevel,
		TransitionTime: tenthsOfSecond,
	})
	if err == nil && i.data[d.Identifier].RequiresPolling {
		time.AfterFunc(PollAfterSetDelay, func() {
			i.attributeMonitor.Poll(ctx, d)
		})
	}

	return err
}

func (i *Implementation) Status(ctx context.Context, device da.Device) (capabilities.LevelStatus, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return capabilities.LevelStatus{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.LevelFlag) {
		return capabilities.LevelStatus{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return capabilities.LevelStatus{CurrentLevel: i.data[d.Identifier].State}, nil
}

func (i *Implementation) LastChangeTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.LevelFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastChangeTime, nil
}

func (i *Implementation) LastUpdateTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.LevelFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastUpdateTime, nil
}
