package zda

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

func TestZigbeeGateway_enableAPSACK(t *testing.T) {
	t.Run("no zigbee device enables supportsAPSAck", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		node := &internalNode{
			mutex: &sync.RWMutex{},
			devices: map[uint8]*internalDevice{
				0: {},
			},
		}

		err := zgw.enableAPSACK(context.Background(), internalNodeEnumeration{node: node})

		assert.NoError(t, err)
		assert.False(t, node.supportsAPSAck)
	})
}
