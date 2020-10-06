package zda

import (
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZdaNodeTable_Nodes(t *testing.T) {
	t.Run("create and then get return a new node", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		node := nt.getNode(ieee)
		assert.Nil(t, node)

		node, wasNew := nt.createNode(ieee)
		assert.NotNil(t, node)
		assert.True(t, wasNew)
		assert.Equal(t, ieee, node.ieeeAddress)

		node = nt.getNode(ieee)
		assert.NotNil(t, node)
		assert.Equal(t, ieee, node.ieeeAddress)
	})

	t.Run("second create is not marked as new", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		node, wasNew := nt.createNode(ieee)
		assert.NotNil(t, node)
		assert.True(t, wasNew)

		node, wasNew = nt.createNode(ieee)
		assert.NotNil(t, node)
		assert.False(t, wasNew)
	})

	t.Run("delete returns false if no node exists", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		present := nt.removeNode(ieee)
		assert.False(t, present)
	})

	t.Run("delete returns true if no node existed and was deleted", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		nt.createNode(ieee)

		present := nt.removeNode(ieee)
		assert.True(t, present)

		node := nt.getNode(ieee)
		assert.Nil(t, node)
	})
}

func TestZdaNodeTable_Devices(t *testing.T) {
	t.Run("getDevice fails if there is no node present for the device", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		device := nt.getDevice(identifier)
		assert.Nil(t, device)
	})

	t.Run("getDevice fails if there is node present for the device, but no device is", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		nt.createNode(ieee)

		device := nt.getDevice(identifier)
		assert.Nil(t, device)
	})

	t.Run("getDevice returns the device if it is present", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		nt.createNode(ieee)

		device, created := nt.createDevice(identifier)
		assert.NotNil(t, device)
		assert.True(t, created)

		device = nt.getDevice(identifier)
		assert.NotNil(t, device)

		assert.Equal(t, identifier.SubIdentifier, device.subidentifier)
	})

	t.Run("createDevice returns nil and false if the node does not exist on a create attempt", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		device, created := nt.createDevice(identifier)
		assert.Nil(t, device)
		assert.False(t, created)
	})

	t.Run("createDevice only returns true on the first create attempt for a device", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		nt.createNode(ieee)

		device, created := nt.createDevice(identifier)
		assert.NotNil(t, device)
		assert.True(t, created)

		device, created = nt.createDevice(identifier)
		assert.NotNil(t, device)
		assert.False(t, created)
	})

	t.Run("createDevice returns nil and false if the node does not exist on a create attempt", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		device := nt.createNextDevice(ieee)
		assert.Nil(t, device)
	})

	t.Run("createNextDevice returns new device with next numerically available subidentifier", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		nt.createNode(ieee)

		nt.createDevice(IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 0,
		})

		nt.createDevice(IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		})

		device := nt.createNextDevice(ieee)
		assert.NotNil(t, device)
		assert.Equal(t, uint8(2), device.subidentifier)
	})

	t.Run("removeDevice returns false if removing a device when the node is not present", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		removed := nt.removeDevice(identifier)
		assert.False(t, removed)
	})

	t.Run("removeDevice returns false if removing a non existent device when the node is present", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		nt.createNode(ieee)

		removed := nt.removeDevice(identifier)
		assert.False(t, removed)
	})

	t.Run("removeDevice returns true when removing a present device", func(t *testing.T) {
		nt := newNodeTable()
		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		identifier := IEEEAddressWithSubIdentifier{
			IEEEAddress:   ieee,
			SubIdentifier: 1,
		}

		nt.createNode(ieee)
		nt.createDevice(identifier)

		removed := nt.removeDevice(identifier)
		assert.True(t, removed)

		device := nt.getDevice(identifier)
		assert.Nil(t, device)
	})
}

func generateNodeTableWithData(devCount uint8) (nodeTable, *internalNode, []*internalDevice) {
	nt := newNodeTable()
	var node *internalNode
	var devices []*internalDevice

	ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

	node, _ = nt.createNode(ieee)
	node.nodeDesc.ManufacturerCode = 0x1234

	for subId := uint8(0); subId < devCount; subId++ {
		endpoint := zigbee.Endpoint(subId)

		dev, _ := nt.createDevice(IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: subId})

		dev.endpoints = []zigbee.Endpoint{endpoint}
		dev.deviceID = uint16(subId)
		dev.deviceVersion = 1

		node.endpoints = append(node.endpoints, endpoint)
		node.endpointDescriptions[endpoint] = zigbee.EndpointDescription{
			Endpoint:       endpoint,
			ProfileID:      zigbee.ProfileHomeAutomation,
			DeviceID:       dev.deviceID,
			DeviceVersion:  dev.deviceVersion,
			InClusterList:  []zigbee.ClusterID{},
			OutClusterList: []zigbee.ClusterID{},
		}

		devices = append(devices, dev)
	}

	return nt, node, devices
}
