package zda

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_node_nextTransactionSequence(t *testing.T) {
	t.Run("iterates through transaction sequences and wraps at end", func(t *testing.T) {
		n := node{
			sequence: make(chan uint8, 3),
		}

		n.sequence <- 1
		n.sequence <- 2
		n.sequence <- 3

		assert.Equal(t, uint8(1), n.nextTransactionSequence())
		assert.Equal(t, uint8(2), n.nextTransactionSequence())
		assert.Equal(t, uint8(3), n.nextTransactionSequence())
		assert.Equal(t, uint8(1), n.nextTransactionSequence())
	})
}

func Test_node_nextDeviceSubIdentifier(t *testing.T) {
	t.Run("finds the next sub identifier that is available", func(t *testing.T) {
		n := node{
			device: map[uint8]*device{0: nil, 1: nil, 2: nil},
		}

		assert.Equal(t, uint8(3), n.nextDeviceSubIdentifier())
	})
}
