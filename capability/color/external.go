package color

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/da/capabilities/color"
	"time"
)

var _ capabilities.Color = (*Implementation)(nil)

const PollAfterSetDelay = 100 * time.Millisecond

func (i *Implementation) ChangeColor(ctx context.Context, device da.Device, convertibleColor color.ConvertibleColor, duration time.Duration) error {
	panic("implement me")
}

func (i *Implementation) ChangeTemperature(ctx context.Context, device da.Device, f float64, duration time.Duration) error {
	panic("implement me")
}

func (i *Implementation) SupportsColor() bool {
	panic("implement me")
}

func (i *Implementation) SupportsTemperature() bool {
	panic("implement me")
}

func (i *Implementation) Status(ctx context.Context, device da.Device) (capabilities.ColorStatus, error) {
	panic("implement me")
}
