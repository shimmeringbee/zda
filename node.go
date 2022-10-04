package zda

import (
	"github.com/shimmeringbee/zigbee"
	"math"
	"sync"
)

type node struct {
	// Immutable data.
	address zigbee.IEEEAddress
	m       *sync.RWMutex

	// Safe data.
	sequence chan uint8

	// Mutable data, obtain lock first.
	device map[uint8]*device
}

func (n *node) nextTransactionSequence() uint8 {
	nextSeq := <-n.sequence
	n.sequence <- nextSeq

	return nextSeq
}

func (n *node) _nextDeviceSubIdentifier() uint8 {
	for i := uint8(0); i < math.MaxUint8; i++ {
		if _, found := n.device[i]; !found {
			return i
		}
	}

	return math.MaxUint8
}
