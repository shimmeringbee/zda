package zda

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestZdaPoller(t *testing.T) {
	t.Run("jobs are called after at least the initial delay, and then called repeatedly", func(t *testing.T) {
		nt, _, devices := generateNodeTableWithData(1)

		poller := zdaPoller{
			nodeTable: nt,
			randLock:  &sync.Mutex{},
		}

		poller.Start()
		defer poller.Stop()

		called := 0

		poller.Add(devices[0].generateIdentifier(), 5*time.Millisecond, func(ctx context.Context, node *internalDevice) bool {
			called++
			return true
		})

		time.Sleep(20 * time.Millisecond)

		assert.GreaterOrEqual(t, called, 1)
	})

	t.Run("jobs are not called if they are not in the node store", func(t *testing.T) {
		nt, node, devices := generateNodeTableWithData(1)
		nt.removeNode(node.ieeeAddress)

		poller := zdaPoller{
			nodeTable: nt,
			randLock:  &sync.Mutex{},
		}

		poller.Start()
		defer poller.Stop()

		called := false

		poller.Add(devices[0].generateIdentifier(), 5*time.Millisecond, func(ctx context.Context, node *internalDevice) bool {
			called = true
			return false
		})

		time.Sleep(10 * time.Millisecond)

		assert.False(t, called)
	})
}
