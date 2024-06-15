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

func (z *ZDA) createNode(addr zigbee.IEEEAddress) (*node, bool) {
	z.nodeLock.Lock()
	defer z.nodeLock.Unlock()

	n, found := z.node[addr]
	if !found {
		n = &node{
			address:        addr,
			m:              &sync.RWMutex{},
			sequence:       makeTransactionSequence(),
			device:         make(map[uint8]*device),
			enumerationSem: semaphore.NewWeighted(1),
		}

		z.node[addr] = n

		z.sectionForNode(n.address)
	}

	return n, !found
}

func (z *ZDA) getNode(addr zigbee.IEEEAddress) *node {
	z.nodeLock.RLock()
	defer z.nodeLock.RUnlock()

	return z.node[addr]
}

func (z *ZDA) removeNode(addr zigbee.IEEEAddress) bool {
	z.nodeLock.Lock()
	defer z.nodeLock.Unlock()

	_, found := z.node[addr]
	if found {
		delete(z.node, addr)
		z.sectionRemoveNode(addr)
	}

	return found
}

func (z *ZDA) getDevice(addr IEEEAddressWithSubIdentifier) *device {
	n := z.getNode(addr.IEEEAddress)

	if n == nil {
		return nil
	}

	n.m.RLock()
	defer n.m.RUnlock()

	return n.device[addr.SubIdentifier]
}

func (z *ZDA) getDevices() []*device {
	z.nodeLock.Lock()
	defer z.nodeLock.Unlock()

	var devices []*device

	for _, n := range z.node {
		devices = append(devices, z.getDevicesOnNode(n)...)
	}

	return devices
}

func (z *ZDA) getDevicesOnNode(n *node) []*device {
	n.m.RLock()
	defer n.m.RUnlock()

	var devices []*device

	for _, d := range n.device {
		devices = append(devices, d)
	}

	return devices
}

func (z *ZDA) createSpecificDevice(n *node, subId uint8) *device {
	n.m.Lock()
	defer n.m.Unlock()

	d := &device{
		address: IEEEAddressWithSubIdentifier{
			IEEEAddress:   n.address,
			SubIdentifier: subId,
		},
		gw:           z,
		n:            n,
		m:            &sync.RWMutex{},
		capabilities: make(map[da.Capability]implcaps.ZDACapability),
	}

	n.device[subId] = d

	z.sectionForDevice(d.address)

	d.eda = &enumeratedDeviceAttachment{
		node:   n,
		device: d,
		ed:     z.ed,

		m: &sync.RWMutex{},
	}

	d.dr = &deviceRemoval{
		node:        n,
		logger:      z.logger,
		nodeRemover: z.provider,
	}

	z.sendEvent(da.DeviceAdded{Device: d})
	z.sendEvent(da.CapabilityAdded{Device: d, Capability: capabilities.EnumerateDeviceFlag})
	z.sendEvent(da.CapabilityAdded{Device: d, Capability: capabilities.DeviceRemovalFlag})

	return d
}

func (z *ZDA) createNextDevice(n *node) *device {
	n.m.Lock()
	subId := n._nextDeviceSubIdentifier()
	n.m.Unlock()

	return z.createSpecificDevice(n, subId)
}

func (z *ZDA) removeDevice(ctx context.Context, addr IEEEAddressWithSubIdentifier) bool {
	n := z.getNode(addr.IEEEAddress)

	if n == nil {
		return false
	}

	n.m.Lock()
	defer n.m.Unlock()

	if d, found := n.device[addr.SubIdentifier]; found {
		d.m.RLock()
		for cf, impl := range d.capabilities {
			z.logger.LogInfo(ctx, "Detaching capability from removed device.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()))
			if err := impl.Detach(ctx, implcaps.DeviceRemoved); err != nil {
				z.logger.LogWarn(ctx, "Error thrown while detaching capability.", logwrap.Datum("Capability", capabilities.StandardNames[cf]), logwrap.Datum("CapabilityImplementation", impl.ImplName()), logwrap.Err(err))
			}

			z.detachCapabilityFromDevice(d, impl)
		}
		d.m.RUnlock()

		z.sendEvent(da.CapabilityRemoved{Device: d, Capability: capabilities.EnumerateDeviceFlag})
		z.sendEvent(da.CapabilityRemoved{Device: d, Capability: capabilities.DeviceRemovalFlag})
		z.sendEvent(da.DeviceRemoved{Device: d})

		delete(n.device, addr.SubIdentifier)
		z.sectionRemoveDevice(d.address)
		return true
	}

	return false
}

func (z *ZDA) attachCapabilityToDevice(d *device, c implcaps.ZDACapability) {
	cF := c.Capability()

	d.capabilities[cF] = c
	z.sectionForDevice(d.address).Section("capability", capabilities.StandardNames[cF])
	z.sendEvent(da.CapabilityAdded{Device: d, Capability: cF})
}

func (z *ZDA) detachCapabilityFromDevice(d *device, c implcaps.ZDACapability) {
	cF := c.Capability()
	if _, found := d.capabilities[cF]; found {
		z.sendEvent(da.CapabilityRemoved{Device: d, Capability: cF})
		z.sectionForDevice(d.address).Section("capability").SectionDelete(capabilities.StandardNames[cF])
		delete(d.capabilities, cF)
	}
}
