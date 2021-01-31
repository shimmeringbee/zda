package color

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/color_control"
	"math"
	"time"
)

var _ capabilities.Color = (*Implementation)(nil)
var _ capabilities.WithLastChangeTime = (*Implementation)(nil)
var _ capabilities.WithLastUpdateTime = (*Implementation)(nil)

const PollAfterSetDelay = 100 * time.Millisecond

func (i *Implementation) ChangeColor(ctx context.Context, device da.Device, outColor color.ConvertibleColor, duration time.Duration) error {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	data := i.data[d.Identifier]

	if !(data.SupportsXY || data.SupportsHueSat) {
		return fmt.Errorf("device does not support color")
	}

	var defaultOutputMode color.NativeColorspace

	if data.SupportsHueSat {
		defaultOutputMode = color.HueSat
	}

	if data.SupportsXY {
		defaultOutputMode = color.XYY
	}

	if data.SupportsHueSat && (outColor.NativeColorspace() == color.HueSat || outColor.NativeColorspace() == color.SRGB) {
		defaultOutputMode = color.HueSat
	}

	if data.SupportsHueSat && outColor.NativeColorspace() == color.XYY {
		defaultOutputMode = color.XYY
	}

	transitionTime := uint16(math.Round(float64(duration / (100 * time.Millisecond))))

	switch defaultOutputMode {
	case color.HueSat:
		h, s, _ := outColor.HSV()

		wireHue := uint8(math.Round((h / 360.0) * 254.0))
		wireSat := uint8(math.Round(s * 254.0))

		return i.supervisor.ZCL().SendCommand(ctx, d, i.data[d.Identifier].Endpoint, zcl.ColorControlId, &color_control.MoveToHueAndSaturation{
			Hue:            wireHue,
			Saturation:     wireSat,
			TransitionTime: transitionTime,
		})
	case color.XYY:
		x, y, _ := outColor.XYY()

		wireX := uint16(math.Round(x * 65536.0))
		wireY := uint16(math.Round(y * 65536.0))

		return i.supervisor.ZCL().SendCommand(ctx, d, i.data[d.Identifier].Endpoint, zcl.ColorControlId, &color_control.MoveToColor{
			ColorX:         wireX,
			ColorY:         wireY,
			TransitionTime: transitionTime,
		})
	}

	return fmt.Errorf("unknown color space selected")
}

func (i *Implementation) ChangeTemperature(ctx context.Context, device da.Device, f float64, duration time.Duration) error {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	if !i.data[d.Identifier].SupportsTemperature {
		return fmt.Errorf("device does not support temperature")
	}

	temp := uint16(math.Round(1000000.0 / f))
	transitionTime := uint16(math.Round(float64(duration / (100 * time.Millisecond))))

	return i.supervisor.ZCL().SendCommand(ctx, d, i.data[d.Identifier].Endpoint, zcl.ColorControlId, &color_control.MoveToColorTemperature{
		ColorTemperatureMireds: temp,
		TransitionTime:         transitionTime,
	})
}

func (i *Implementation) SupportsColor(ctx context.Context, device da.Device) (bool, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return false, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return false, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	state := i.data[d.Identifier]
	return state.SupportsXY || state.SupportsHueSat, nil

}

func (i *Implementation) SupportsTemperature(ctx context.Context, device da.Device) (bool, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return false, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return false, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	state := i.data[d.Identifier]
	return state.SupportsTemperature, nil
}

func (i *Implementation) Status(ctx context.Context, device da.Device) (capabilities.ColorStatus, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(device)
	if !found {
		return capabilities.ColorStatus{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return capabilities.ColorStatus{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.stateToColorStatus(i.data[d.Identifier].State), nil
}

func (i *Implementation) LastChangeTime(ctx context.Context, dad da.Device) (time.Time, error) {
	d, found := i.supervisor.DeviceLookup().ByDA(dad)
	if !found {
		return time.Time{}, da.DeviceDoesNotBelongToGatewayError
	} else if !d.HasCapability(capabilities.ColorFlag) {
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
	} else if !d.HasCapability(capabilities.ColorFlag) {
		return time.Time{}, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	return i.data[d.Identifier].LastUpdateTime, nil
}
