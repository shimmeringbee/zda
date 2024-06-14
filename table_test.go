package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zda/implcaps/generic"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_gateway_createNode(t *testing.T) {
	t.Run("creates a new node if non exists", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_, found := g.node[addr]
		assert.False(t, found)

		n, created := g.createNode(addr)
		assert.NotNil(t, n)
		assert.Equal(t, addr, n.address)
		assert.True(t, created)

		nf, found := g.node[addr]
		assert.True(t, found)
		assert.Equal(t, n, nf)
	})

	t.Run("does not create a new node if already exists", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		n.sequence = nil

		n, created := g.createNode(addr)
		assert.Nil(t, n.sequence)
		assert.False(t, created)
	})
}

func Test_gateway_getNode(t *testing.T) {
	t.Run("returns node if it is present", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		assert.Equal(t, n, g.getNode(addr))
	})

	t.Run("returns nil if note is not present", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		assert.Nil(t, g.getNode(addr))
	})
}

func Test_gateway_removeNode(t *testing.T) {
	t.Run("returns true and removes node if address is present", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_, _ = g.createNode(addr)
		assert.True(t, g.removeNode(addr))
	})

	t.Run("returns false if removing non existent address", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		assert.False(t, g.removeNode(addr))
	})
}

func Test_gateway_createNextDevice(t *testing.T) {
	t.Run("creates a new device on a node with the next free sub identifier", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		assert.Equal(t, n, d.n)

		assert.Equal(t, addr, d.address.IEEEAddress)
		assert.Equal(t, uint8(0), d.address.SubIdentifier)
		assert.Equal(t, g, d.gw)

		assert.NotNil(t, d.eda)
		assert.NotNil(t, d.dr)

		assert.Contains(t, d.Capabilities(), capabilities.EnumerateDeviceFlag)
		assert.Contains(t, d.Capabilities(), capabilities.DeviceRemovalFlag)

		d = g.createNextDevice(n)

		assert.Equal(t, addr, d.address.IEEEAddress)
		assert.Equal(t, uint8(1), d.address.SubIdentifier)

		events := drainEvents(g)
		assert.Len(t, events, 6)
		assert.IsType(t, da.DeviceAdded{}, events[0])
		assert.IsType(t, da.CapabilityAdded{}, events[1])
		assert.IsType(t, da.CapabilityAdded{}, events[2])
		assert.IsType(t, da.DeviceAdded{}, events[3])
		assert.IsType(t, da.CapabilityAdded{}, events[4])
		assert.IsType(t, da.CapabilityAdded{}, events[5])
	})
}

func Test_gateway_getDevice(t *testing.T) {
	t.Run("if a device is present it will be returned, and found will be true", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		dF := g.getDevice(d.address)
		assert.Equal(t, d, dF)
	})

	t.Run("if a device is missing nil will be returned, and found will be false", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		_, _ = g.createNode(addr)

		dF := g.getDevice(IEEEAddressWithSubIdentifier{
			IEEEAddress:   addr,
			SubIdentifier: 0,
		})

		assert.Nil(t, dF)
	})
}

func Test_gateway_getDevices(t *testing.T) {
	t.Run("returns all devices registered", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		addr1 := zigbee.GenerateLocalAdministeredIEEEAddress()
		n1, _ := g.createNode(addr1)
		d1 := g.createNextDevice(n1)

		addr2 := zigbee.GenerateLocalAdministeredIEEEAddress()
		n2, _ := g.createNode(addr2)
		d2 := g.createNextDevice(n2)

		devices := g.getDevices()
		assert.Len(t, devices, 2)
		assert.Contains(t, devices, d1)
		assert.Contains(t, devices, d2)
	})
}

func Test_gateway_getDevicesOnNode(t *testing.T) {
	t.Run("returns all devices registered on the provided node", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)
		addr1 := zigbee.GenerateLocalAdministeredIEEEAddress()
		n1, _ := g.createNode(addr1)
		d1 := g.createNextDevice(n1)
		d2 := g.createNextDevice(n1)

		addr2 := zigbee.GenerateLocalAdministeredIEEEAddress()
		n2, _ := g.createNode(addr2)
		d3 := g.createNextDevice(n2)

		devices := g.getDevicesOnNode(n1)
		assert.Len(t, devices, 2)
		assert.Contains(t, devices, d1)
		assert.Contains(t, devices, d2)
		assert.NotContains(t, devices, d3)
	})
}

func drainEvents(g *gateway) []any {
	events := make([]any, len(g.events))

	for i := range len(g.events) {
		events[i] = <-g.events
	}

	return events
}

func Test_gateway_removeDevice(t *testing.T) {
	t.Run("removes a device from a node, and returns true", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()
		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		assert.NotNil(t, g.getDevice(d.address))
		assert.True(t, g.removeDevice(context.Background(), d.address))
		assert.Nil(t, g.getDevice(d.address))

		events := drainEvents(g)
		assert.Len(t, events, 6)
		assert.IsType(t, da.DeviceAdded{}, events[0])
		assert.IsType(t, da.CapabilityAdded{}, events[1])
		assert.IsType(t, da.CapabilityAdded{}, events[2])
		assert.IsType(t, da.CapabilityRemoved{}, events[3])
		assert.IsType(t, da.CapabilityRemoved{}, events[4])
		assert.IsType(t, da.DeviceRemoved{}, events[5])
	})

	t.Run("returns false if device can't be found on node", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()
		_, _ = g.createNode(addr)

		assert.False(t, g.removeDevice(context.Background(), IEEEAddressWithSubIdentifier{
			IEEEAddress:   addr,
			SubIdentifier: 0,
		}))

		select {
		case _ = <-g.events:
			t.Error("non existent device removal should not have emitted event")
		default:
		}
	})
}

func Test_gateway_attachCapabilityToDevice(t *testing.T) {
	t.Run("attaches capability to device and emits event", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		_ = drainEvents(g)

		c := generic.NewProductInformation()
		g.attachCapabilityToDevice(d, c)

		assert.Contains(t, d.capabilities, capabilities.ProductInformationFlag)

		events := drainEvents(g)
		assert.Len(t, events, 1)
		assert.IsType(t, da.CapabilityAdded{}, events[0])

		assert.True(t, g.sectionForDevice(d.address).Section("capability").SectionExists(capabilities.StandardNames[capabilities.ProductInformationFlag]))
	})
}

func Test_gateway_detachCapabilityFromDevice(t *testing.T) {
	t.Run("detaches a capability from device and emits event", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		c := generic.NewProductInformation()
		g.attachCapabilityToDevice(d, c)

		assert.True(t, g.sectionForDevice(d.address).Section("capability").SectionExists(capabilities.StandardNames[capabilities.ProductInformationFlag]))

		_ = drainEvents(g)

		g.detachCapabilityFromDevice(d, c)

		assert.NotContains(t, d.capabilities, capabilities.ProductInformationFlag)

		events := drainEvents(g)
		assert.Len(t, events, 1)
		assert.IsType(t, da.CapabilityRemoved{}, events[0])

		assert.False(t, g.sectionForDevice(d.address).Section("capability").SectionExists(capabilities.StandardNames[capabilities.ProductInformationFlag]))
	})

	t.Run("does nothing if called for unattached capability", func(t *testing.T) {
		g := New(context.Background(), memory.New(), nil, nil).(*gateway)
		g.events = make(chan any, 0xffff)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()

		n, _ := g.createNode(addr)
		d := g.createNextDevice(n)

		c := generic.NewProductInformation()

		_ = drainEvents(g)

		g.detachCapabilityFromDevice(d, c)

		assert.Len(t, g.events, 0)
	})
}
