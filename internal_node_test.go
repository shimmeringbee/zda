package zda

import (
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func Test_internalNode_nextTransactionSequence(t *testing.T) {
	t.Run("receives the next transaction sequence", func(t *testing.T) {
		ieeeAddress := zigbee.IEEEAddress(0x0102030405060708)
		iNode := internalNode{
			ieeeAddress:          ieeeAddress,
			mutex:                &sync.RWMutex{},
			devices:              map[uint8]*internalDevice{},
			transactionSequences: make(chan uint8, 3),
		}

		iNode.transactionSequences <- 1
		iNode.transactionSequences <- 2
		iNode.transactionSequences <- 3

		assert.Equal(t, uint8(1), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(2), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(3), iNode.nextTransactionSequence())
		assert.Equal(t, uint8(1), iNode.nextTransactionSequence())
	})
}
