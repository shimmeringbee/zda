package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
	"sync"
	"testing"
)

func Test_enumerateDevice_startEnumeration(t *testing.T) {
	t.Run("returns an error if the node is already being enumerated", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard())}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		n.enumerationSem.TryAcquire(1)
		err := ed.startEnumeration(context.Background(), n)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "enumeration already in progress")
	})

	t.Run("returns nil if node is not being enumerated, and marks the node in progress", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard())}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		err := ed.startEnumeration(context.Background(), n)
		assert.Nil(t, err)
		assert.False(t, n.enumerationSem.TryAcquire(1))
	})
}
