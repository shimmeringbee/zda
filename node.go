package zda

import (
	"github.com/shimmeringbee/zigbee"
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
