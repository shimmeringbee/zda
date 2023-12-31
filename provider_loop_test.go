package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
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

		g := New(context.Background(), mp, nil).(*gateway)
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
	})
}

func Test_gateway_receiveNodeLeaveEvent(t *testing.T) {
	t.Run("node leave event will remove the node from the node table, removing any devices", func(t *testing.T) {
		g := New(context.Background(), nil, nil).(*gateway)
		g.WithLogWrapLogger(logwrap.New(discard.Discard()))
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		_ = g.createNextDevice(n)

		g.receiveNodeLeaveEvent(zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress: addr,
			},
		})

		assert.Nil(t, g.getNode(addr))
		assert.Empty(t, n.device)

		d := g.getDevice(IEEEAddressWithSubIdentifier{
			IEEEAddress:   addr,
			SubIdentifier: 0,
		})

		assert.Nil(t, d)
	})
}
