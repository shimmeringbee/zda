package zda

import (
	"github.com/shimmeringbee/zigbee"
	"golang.org/x/sync/semaphore"
	"math"
	"sync"
)

type productData struct {
	manufacturer string
	product      string
	version      string
	serial       string
}

type endpointDetails struct {
	description        zigbee.EndpointDescription
	productInformation productData
}

type inventory struct {
	description *zigbee.NodeDescription
	endpoints   map[zigbee.Endpoint]endpointDetails
}

type node struct {
	// Immutable data.
	address zigbee.IEEEAddress
	m       *sync.RWMutex

	// Thread safe data.
	sequence chan uint8

	// Mutable data, obtain lock first.
	device map[uint8]*device

	useAPSAck bool

	// Enumeration data.
	enumerationSem    *semaphore.Weighted
	originalInventory *inventory
	resolvedInventory *inventory
}

func makeTransactionSequence() chan uint8 {
	ch := make(chan uint8, math.MaxUint8)

	for i := uint8(0); i < math.MaxUint8; i++ {
		ch <- i
	}

	return ch
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
