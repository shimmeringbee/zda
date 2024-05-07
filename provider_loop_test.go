package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"testing"
)

func Test_gateway_receiveNodeJoinEvent(t *testing.T) {
	t.Run("node join event will add the new node to the node table, and introduce a base device", func(t *testing.T) {
		mp := &zigbee.MockProvider{}
		mp.On("QueryNodeDescription", mock.Anything, mock.Anything).Return(zigbee.NodeDescription{}, io.EOF).Maybe()
		defer mp.AssertExpectations(t)

		g := New(context.Background(), memory.New(), mp, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		g.WithLogWrapLogger(logwrap.New(discard.Discard()))
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		called := false
		g.callbacks.Add(func(ctx context.Context, join nodeJoin) error {
			called = true
			return nil
		})

		g.receiveNodeJoinEvent(zigbee.NodeJoinEvent{
			Node: zigbee.Node{
				IEEEAddress: addr,
			},
		})

		n := g.getNode(addr)

		assert.NotNil(t, n)
		assert.Equal(t, addr, n.address)

		d := g.getDevice(IEEEAddressWithSubIdentifier{
			IEEEAddress:   addr,
			SubIdentifier: 0,
		})

		assert.NotNil(t, d)
		assert.True(t, called)

		assert.Contains(t, g.nodeListFromPersistence(), addr)
		assert.Contains(t, g.deviceListFromPersistence(addr), d.address)
	})
}

func Test_gateway_receiveNodeLeaveEvent(t *testing.T) {
	t.Run("node leave event will remove the node from the node table, removing any devices", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		g.WithLogWrapLogger(logwrap.New(discard.Discard()))
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		assert.Contains(t, g.nodeListFromPersistence(), addr)
		assert.Contains(t, g.deviceListFromPersistence(addr), d.address)

		g.receiveNodeLeaveEvent(zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress: addr,
			},
		})

		assert.Nil(t, g.getNode(addr))
		assert.Empty(t, n.device)

		assert.NotContains(t, g.nodeListFromPersistence(), addr)
		assert.NotContains(t, g.deviceListFromPersistence(addr), d.address)

		d = g.getDevice(d.address)

		assert.Nil(t, d)
	})
}
