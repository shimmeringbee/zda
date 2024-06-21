package device_workaround

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"time"
)

var _ implcaps.ZDACapability = (*Implementation)(nil)

func NewDeviceWorkaround(zi implcaps.ZDAInterface) *Implementation {
	return &Implementation{zi: zi}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	am attribute.Monitor
	zi implcaps.ZDAInterface
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.DeviceWorkaroundFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.DeviceWorkaroundFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s

	i.am = i.zi.NewAttributeMonitor()
	i.am.Init(s.Section("AttributeMonitor", "ZCLVersion"), d, i.update)
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

	reporting := attribute.ReportingConfig{
		Mode:             attribute.AttemptConfigureReporting,
		MinimumInterval:  1 * time.Minute,
		MaximumInterval:  2 * time.Minute,
		ReportableChange: uint(0),
	}

	polling := attribute.PollingConfig{
		Mode: attribute.NeverPoll,
	}

	if err := i.am.Attach(ctx, endpoint, zcl.BasicId, basic.ZCLVersion, zcl.TypeUnsignedInt8, reporting, polling); err != nil {
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
	return "ProprietaryTiRouterDeviceWorkaround"
}

func (i *Implementation) update(_ zcl.AttributeID, _ zcl.AttributeDataTypeValue) {

}
