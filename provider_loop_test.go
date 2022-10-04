package zda

import (
	"context"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_gateway_receiveNodeJoinEvent(t *testing.T) {
	t.Run("node join event will add the new node to the node table", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		g.receiveNodeJoinEvent(zigbee.NodeJoinEvent{
			Node: zigbee.Node{
				IEEEAddress: addr,
			},
		})

		n := g.getNode(addr)
		assert.NotNil(t, n)
		assert.Equal(t, addr, n.address)
	})
}

func Test_gateway_receiveNodeLeaveEvent(t *testing.T) {
	t.Run("node leave event will remove the node from the node table", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_ = g.createNode(addr)

		g.receiveNodeLeaveEvent(zigbee.NodeLeaveEvent{
			Node: zigbee.Node{
				IEEEAddress: addr,
			},
		})

		n := g.getNode(addr)
		assert.Nil(t, n)
	})
}
