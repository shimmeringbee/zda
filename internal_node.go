package zda

import (
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type internalNode struct {
	// Immutable, no locking required.
	ieeeAddress zigbee.IEEEAddress
	mutex       *sync.RWMutex

	// Mutable, locking must be obtained first.
	devices map[uint8]*internalDevice

	nodeDesc             zigbee.NodeDescription
	endpoints            []zigbee.Endpoint
	endpointDescriptions map[zigbee.Endpoint]zigbee.EndpointDescription

	enumerating          bool
	transactionSequences chan uint8
	supportsAPSAck       bool
}

func (n *internalNode) getDevices() []*internalDevice {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	var devices []*internalDevice

	for _, device := range n.devices {
		devices = append(devices, device)
	}

	return devices
}

func (n *internalNode) nextTransactionSequence() uint8 {
	nextSeq := <-n.transactionSequences
	n.transactionSequences <- nextSeq

	return nextSeq
}
