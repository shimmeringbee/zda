package on_off

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
)

func (i *Implementation) AddedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[d.Identifier]; !found {
		i.data[d.Identifier] = Data{}
	}

	return nil
}

func (i *Implementation) RemovedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	i.attributeMonitor.Detach(ctx, d)
	delete(i.data, d.Identifier)

	return nil
}

func selectEndpoint(found []zigbee.Endpoint, device map[zigbee.Endpoint]zigbee.EndpointDescription) zigbee.Endpoint {
	if len(found) > 0 {
		return found[0]
	}

	if len(device) > 0 {
		for endpoint := range device {
			return endpoint
		}
	}

	return 0
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	cfg := i.supervisor.DeviceConfig().Get(d, i.KeyName())

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.OnOffId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	if !hasCapability {
		i.attributeMonitor.Detach(ctx, d)

		i.datalock.Lock()

		i.data[d.Identifier] = Data{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.TemperatureSensorFlag)
	} else {
		var data Data
		data.Endpoint = zigbee.Endpoint(cfg.Int("Endpoint", int(selectEndpoint(endpoints, d.Endpoints))))

		requiresPolling, _ := i.attributeMonitor.Attach(ctx, d, data.Endpoint, nil)

		data.RequiresPolling = requiresPolling

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.TemperatureSensorFlag)
	}

	return nil
}

func (i *Implementation) setState(d zda.Device, s bool) {
	i.datalock.Lock()

	data := i.data[d.Identifier]

	if data.State != s {
		data.State = s
		i.data[d.Identifier] = data

		i.supervisor.DAEventSender().Send(capabilities.OnOffState{
			Device: i.supervisor.ComposeDADevice().Compose(d),
			State:  s,
		})
	}

	i.datalock.Unlock()
}
