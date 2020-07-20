package zda

import (
	"context"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestZdaPoller(t *testing.T) {
	t.Run("jobs are called after at least the initial delay, and then called repeatedly", func(t *testing.T) {
		node := &internalNode{ieeeAddress: zigbee.GenerateLocalAdministeredIEEEAddress()}

		mockNodeStore := mockNodeStore{}
		mockNodeStore.On("getNode", node.ieeeAddress).Return(node, true)

		poller := zdaPoller{
			nodeStore: &mockNodeStore,
		}

		poller.Start()
		defer poller.Stop()

		called := 0

		poller.AddNode(node, 5*time.Millisecond, func(ctx context.Context, node *internalNode) {
			called++
		})

		time.Sleep(20 * time.Millisecond)

		assert.Greater(t, called, 1)
	})

	t.Run("jobs are not called if they are not in the node store", func(t *testing.T) {
		node := &internalNode{ieeeAddress: zigbee.GenerateLocalAdministeredIEEEAddress()}

		mockNodeStore := mockNodeStore{}
		mockNodeStore.On("getNode", node.ieeeAddress).Return(&internalNode{}, false)

		poller := zdaPoller{
			nodeStore: &mockNodeStore,
		}

		poller.Start()
		defer poller.Stop()

		called := false

		poller.AddNode(node, 5*time.Millisecond, func(ctx context.Context, node *internalNode) {
			called = true
		})

		time.Sleep(10 * time.Millisecond)

		assert.False(t, called)
	})
}
