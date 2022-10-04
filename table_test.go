package zda

import (
	"context"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_gateway_createNode(t *testing.T) {
	t.Run("creates a new node if non exists", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_, found := g.node[addr]
		assert.False(t, found)

		n := g.createNode(addr)
		assert.NotNil(t, n)
		assert.Equal(t, addr, n.address)

		nf, found := g.node[addr]
		assert.True(t, found)
		assert.Equal(t, n, nf)
	})

	t.Run("does not create a new node if already exists", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n := g.createNode(addr)
		n.sequence = nil

		n = g.createNode(addr)
		assert.Nil(t, n.sequence)
	})
}

func Test_gateway_getNode(t *testing.T) {
	t.Run("returns node if it is present", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n := g.createNode(addr)
		assert.Equal(t, n, g.getNode(addr))
	})

	t.Run("returns nil if note is not present", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		assert.Nil(t, g.getNode(addr))
	})
}

func Test_gateway_removeNode(t *testing.T) {
	t.Run("returns true and removes node if address is present", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_ = g.createNode(addr)
		assert.True(t, g.removeNode(addr))
	})

	t.Run("returns false if removing non existent address", func(t *testing.T) {
		g := New(context.Background(), nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		assert.False(t, g.removeNode(addr))
	})
}
