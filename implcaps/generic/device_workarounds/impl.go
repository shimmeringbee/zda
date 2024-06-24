package device_workaround

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"strings"
	"sync"
)

var _ implcaps.ZDACapability = (*Implementation)(nil)
var _ capabilities.DeviceWorkarounds = (*Implementation)(nil)

func NewDeviceWorkaround(zi implcaps.ZDAInterface) *Implementation {
	return &Implementation{zi: zi, m: &sync.RWMutex{}}
}

type Implementation struct {
	s  persistence.Section
	d  da.Device
	zi implcaps.ZDAInterface

	m                  *sync.RWMutex
	workaroundsEnabled []string
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.DeviceWorkaroundsFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.DeviceWorkaroundsFlag]
}

func (i *Implementation) Init(d da.Device, s persistence.Section) {
	i.d = d
	i.s = s
}

func (i *Implementation) Load(ctx context.Context) (bool, error) {
	i.m.Lock()
	defer i.m.Unlock()

	i.workaroundsEnabled = i.s.Section("Workarounds").SectionKeys()

	for _, workaround := range i.workaroundsEnabled {
		if err := i.loadWorkaround(ctx, workaround); err != nil {
			return false, err
		} else {
			i.workaroundsEnabled = append(i.workaroundsEnabled, "ZCLReportingKeepAlive")
		}
	}

	return true, nil
}

func (i *Implementation) Enumerate(ctx context.Context, m map[string]any) (bool, error) {
	var workarounds []string

	for k, _ := range m {
		if strings.HasPrefix(k, "Enable") {
			workarounds = append(workarounds, k)
		}
	}

	i.m.Lock()
	i.workaroundsEnabled = workarounds
	i.m.Unlock()

	for _, workaround := range workarounds {
		if err := i.enumerateWorkaround(ctx, m, workaround); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (i *Implementation) Detach(ctx context.Context, detachType implcaps.DetachType) error {
	i.m.Lock()
	defer i.m.Unlock()

	for _, workaround := range i.workaroundsEnabled {
		if err := i.detachWorkaround(ctx, detachType, workaround); err != nil {
			return err
		}
	}

	return nil
}

func (i *Implementation) ImplName() string {
	return "GenericDeviceWorkarounds"
}

func (i *Implementation) Enabled(_ context.Context) ([]string, error) {
	i.m.RLock()
	defer i.m.RUnlock()

	return i.workaroundsEnabled, nil
}

func (i *Implementation) loadWorkaround(ctx context.Context, workaround string) error {
	return nil
}

func (i *Implementation) detachWorkaround(ctx context.Context, detachType implcaps.DetachType, workaround string) error {
	return nil
}

func (i *Implementation) enumerateWorkaround(ctx context.Context, m map[string]any, workaround string) error {
	switch workaround {
	case "EnableZCLReportingKeepAlive":
		return i.enumerateZCLReportingKeepAlive(ctx, m)
	}

	return nil
}

func (i *Implementation) enumerateZCLReportingKeepAlive(ctx context.Context, m map[string]any) error {
	remoteEndpoint := implcaps.Get(m, "ZigbeeEndpoint", zigbee.Endpoint(1))

	ieeeAddress, localEndpoint, ack, seq := i.zi.TransmissionLookup(i.d, zigbee.ProfileHomeAutomation)

	if err := i.zi.NodeBinder().BindNodeToController(ctx, ieeeAddress, localEndpoint, remoteEndpoint, zcl.BasicId); err != nil {
		return err
	}

	if err := i.zi.ZCLCommunicator().ConfigureReporting(ctx, ieeeAddress, ack, zcl.BasicId, zigbee.NoManufacturer, localEndpoint, remoteEndpoint, seq, basic.ZCLVersion, zcl.TypeUnsignedInt8, uint16(60), uint16(240), uint(0)); err != nil {
		return err
	}

	i.s.Section("Workarounds", "ZCLReportingKeepAlive")

	i.m.Lock()
	defer i.m.Unlock()

	i.workaroundsEnabled = append(i.workaroundsEnabled, "ZCLReportingKeepAlive")

	return nil
}
