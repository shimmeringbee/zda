package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCapabilityManager_initSupervisor_DeviceLookup(t *testing.T) {
	t.Run("returns false if gateway doesn't match", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr}

		_, ok := s.DeviceLookup().ByDA(daDevice)
		assert.False(t, ok)
	})

	t.Run("returns false if gateway does match, but isn't found in the node table", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr, DeviceGateway: zgw}

		_, ok := s.DeviceLookup().ByDA(daDevice)
		assert.False(t, ok)
	})

	t.Run("returns true if gateway does match and is found, device details match", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw, nodeTable: newNodeTable()}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		daDevice := da.BaseDevice{DeviceIdentifier: addr, DeviceGateway: zgw}

		iN, _ := m.nodeTable.createNode(addr.IEEEAddress)
		iD, _ := m.nodeTable.createDevice(addr)
		iD.capabilities = []da.Capability{0x0001}

		iN.endpoints = []zigbee.Endpoint{0x01, 0x02}
		iN.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint: 0x01,
			},
			0x02: {
				Endpoint: 0x02,
			},
		}

		iD.endpoints = []zigbee.Endpoint{0x02}

		expectedEndpoints := map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x02: {
				Endpoint: 0x02,
			},
		}

		d, ok := s.DeviceLookup().ByDA(daDevice)
		assert.True(t, ok)

		assert.Equal(t, iD.capabilities, d.Capabilities)
		assert.Equal(t, addr, d.Identifier)
		assert.Equal(t, expectedEndpoints, d.Endpoints)
	})
}
