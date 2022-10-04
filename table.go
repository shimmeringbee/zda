package zda

import (
	"github.com/shimmeringbee/zigbee"
	"math"
	"sync"
)

func (g *gateway) createNode(addr zigbee.IEEEAddress) *node {
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

	return n
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
