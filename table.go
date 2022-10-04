package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"math"
	"sync"
)

func (g *gateway) createNode(addr zigbee.IEEEAddress) (*node, bool) {
	g.nodeLock.Lock()
	defer g.nodeLock.Unlock()

	n, found := g.node[addr]
	if !found {
		n = &node{
			address:  addr,
			m:        &sync.RWMutex{},
			sequence: make(chan uint8, math.MaxUint8),
			device:   make(map[uint8]*device),
		}

		for s := uint8(0); s < math.MaxUint8; s++ {
			n.sequence <- s
		}

		g.node[addr] = n
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
	}

	return found
}

func (g *gateway) createNextDevice(n *node) *device {
	n.m.Lock()
	defer n.m.Unlock()

	subId := n._nextDeviceSubIdentifier()

	return g._createDevice(n, IEEEAddressWithSubIdentifier{
		IEEEAddress:   n.address,
		SubIdentifier: subId,
	})
}

func (g *gateway) _createDevice(n *node, addr IEEEAddressWithSubIdentifier) *device {
	d := &device{
		address:      addr,
		gw:           g,
		m:            &sync.RWMutex{},
		capabilities: []da.Capability{},
	}

	n.device[addr.SubIdentifier] = d
	return d
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

func (g *gateway) removeDevice(addr IEEEAddressWithSubIdentifier) bool {
	n := g.getNode(addr.IEEEAddress)

	if n == nil {
		return false
	}

	n.m.Lock()
	defer n.m.Unlock()

	if _, found := n.device[addr.SubIdentifier]; found {
		delete(n.device, addr.SubIdentifier)
		return true
	}

	return false
}
