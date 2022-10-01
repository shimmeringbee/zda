package zda

import (
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type node struct {
	// Immutable data.
	address zigbee.IEEEAddress
	m       *sync.RWMutex

	// Mutable data, obtain lock first.
	devices map[uint8]*device
}
