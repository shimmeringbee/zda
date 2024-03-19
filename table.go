package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
	"golang.org/x/sync/semaphore"
	"sync"
)

func (g *gateway) createNode(addr zigbee.IEEEAddress) (*node, bool) {
	g.nodeLock.Lock()
	defer g.nodeLock.Unlock()

	n, found := g.node[addr]
	if !found {
		n = &node{
			address:        addr,
			m:              &sync.RWMutex{},
			sequence:       makeTransactionSequence(),
			device:         make(map[uint8]*device),
			enumerationSem: semaphore.NewWeighted(1),
		}

		g.node[addr] = n

		g.sectionForNode(n.address)
	}

	return n, !found
}

func (g *gateway) getNode(addr zigbee.IEEEAddress) *node {
	g.nodeLock.RLock()
	defer g.nodeLock.RUnlock()

	return g.node[addr]
}

func (g *gateway) removeNode(addr zigbee.IEEEAddress) bool {
	g.nodeLock.Lock()
	defer g.nodeLock.Unlock()

	_, found := g.node[addr]
	if found {
		delete(g.node, addr)
		g.sectionRemoveNode(addr)
	}

	return found
}

func (g *gateway) getDevice(addr IEEEAddressWithSubIdentifier) *device {
	n := g.getNode(addr.IEEEAddress)

	if n == nil {
		return nil
	}

	n.m.RLock()
	defer n.m.RUnlock()

	return n.device[addr.SubIdentifier]
}

func (g *gateway) getDevices() []*device {
	g.nodeLock.Lock()
	defer g.nodeLock.Unlock()

	var devices []*device

	for _, n := range g.node {
		devices = append(devices, g.getDevicesOnNode(n)...)
	}

	return devices
}

func (g *gateway) getDevicesOnNode(n *node) []*device {
	n.m.RLock()
	defer n.m.RUnlock()

	var devices []*device

	for _, d := range n.device {
		devices = append(devices, d)
	}

	return devices
}

func (g *gateway) createSpecificDevice(n *node, subId uint8) *device {
	n.m.Lock()
	defer n.m.Unlock()

	d := &device{
		address: IEEEAddressWithSubIdentifier{
			IEEEAddress:   n.address,
			SubIdentifier: subId,
		},
		gw:           g,
		n:            n,
		m:            &sync.RWMutex{},
		capabilities: make(map[da.Capability]implcaps.ZDACapability),
	}

	n.device[subId] = d

	g.sectionForDevice(d.address)

	d.eda = &enumeratedDeviceAttachment{
		node:   n,
		device: d,
		ed:     g.ed,

		m: &sync.RWMutex{},
	}

	d.dr = &deviceRemoval{
		node:        n,
		logger:      g.logger,
		nodeRemover: g.provider,
	}

	g.sendEvent(da.DeviceAdded{Device: d})
	g.sendEvent(da.CapabilityAdded{Device: d, Capability: capabilities.EnumerateDeviceFlag})
	g.sendEvent(da.CapabilityAdded{Device: d, Capability: capabilities.DeviceRemovalFlag})

	return d
}

func (g *gateway) createNextDevice(n *node) *device {
	n.m.Lock()
	subId := n._nextDeviceSubIdentifier()
	n.m.Unlock()

	return g.createSpecificDevice(n, subId)
}

func (g *gateway) removeDevice(ctx context.Context, addr IEEEAddressWithSubIdentifier) bool {
	n := g.getNode(addr.IEEEAddress)

	if n == nil {
		return false
	}

	n.m.Lock()
	defer n.m.Unlock()

	if d, found := n.device[addr.SubIdentifier]; found {
		d.m.RLock()
		for cf, impl := range d.capabilities {
			g.logger.LogInfo(ctx, "Detaching capability from removed device.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()))
			if err := impl.Detach(ctx, implcaps.DeviceRemoved); err != nil {
				g.logger.LogWarn(ctx, "Error thrown while detaching capability.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()), logwrap.Err(err))
			}

			g.detachCapabilityFromDevice(d, impl)
		}
		d.m.RUnlock()

		g.sendEvent(da.CapabilityRemoved{Device: d, Capability: capabilities.EnumerateDeviceFlag})
		g.sendEvent(da.CapabilityRemoved{Device: d, Capability: capabilities.DeviceRemovalFlag})
		g.sendEvent(da.DeviceRemoved{Device: d})

		delete(n.device, addr.SubIdentifier)
		g.sectionRemoveDevice(d.address)
		return true
	}

	return false
}

func (g *gateway) attachCapabilityToDevice(d *device, c implcaps.ZDACapability) {
	cF := c.Capability()

	d.capabilities[cF] = c
	g.sectionForDevice(d.address).Section("capability", capabilities.StandardNames[cF])
	g.sendEvent(da.CapabilityAdded{Device: d, Capability: cF})
}

func (g *gateway) detachCapabilityFromDevice(d *device, c implcaps.ZDACapability) {
	cF := c.Capability()
	if _, found := d.capabilities[cF]; found {
		g.sendEvent(da.CapabilityRemoved{Device: d, Capability: cF})
		g.sectionForDevice(d.address).Section("capability").Delete(capabilities.StandardNames[cF])
		delete(d.capabilities, cF)
	}
}
