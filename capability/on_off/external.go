package on_off

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"time"
)

var _ capabilities.OnOff = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)

const PollAfterSetDelay = 100 * time.Millisecond

func (i *Implementation) On(ctx context.Context, dad da.Device) error {
	return i.cmd(ctx, dad, &onoff.On{})
}

func (i *Implementation) Off(ctx context.Context, dad da.Device) error {
	return i.cmd(ctx, dad, &onoff.Off{})
}

func (i *Implementation) cmd(ctx context.Context, dad da.Device, cmd interface{}) error {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.OnOffFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	err := i.supervisor.ZCL().SendCommand(ctx, d, i.data[d.Identifier].Endpoint, zcl.OnOffId, cmd)
	if err == nil && i.data[d.Identifier].RequiresPolling {
		time.AfterFunc(PollAfterSetDelay, func() {
			i.attributeMonitor.Poll(ctx, d)
		})
	}

	return err
}

func (i *Implementation) Status(ctx context.Context, dad da.Device) (bool, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return false, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.OnOffFlag) {
		return false, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].State, nil
}

func (i *Implementation) LastChangeTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.OnOffFlag) {
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
	} else if !d.HasCapability(capabilities.OnOffFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastUpdateTime, nil
}
