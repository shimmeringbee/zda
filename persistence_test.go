package zda

import (
	"context"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_gateway_deviceListFromPersistence(t *testing.T) {
	t.Run("multiple devices are returned", func(t *testing.T) {
		zgw := New(context.Background(), memory.New(), nil, nil)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		deviceOne := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 1}
		deviceTwo := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 2}

		zgw.sectionForDevice(deviceOne)
		zgw.sectionForDevice(deviceTwo)

		devices := zgw.deviceListFromPersistence(ieee)

		assert.Contains(t, devices, deviceOne)
		assert.Contains(t, devices, deviceTwo)
	})
}

func Test_gateway_nodeListFromPersistence(t *testing.T) {
	t.Run("multiple nodes are returned", func(t *testing.T) {
		zgw := New(context.Background(), memory.New(), nil, nil)

		ieeeOne := zigbee.GenerateLocalAdministeredIEEEAddress()
		ieeeTwo := zigbee.GenerateLocalAdministeredIEEEAddress()

		zgw.sectionForNode(ieeeOne)
		zgw.sectionForNode(ieeeTwo)

		nodes := zgw.nodeListFromPersistence()

		assert.Contains(t, nodes, ieeeOne)
		assert.Contains(t, nodes, ieeeTwo)
	})
}

func Test_gateway_sectionRemoveNode(t *testing.T) {
	t.Run("nodes are removed", func(t *testing.T) {
		zgw := New(context.Background(), memory.New(), nil, nil)

		ieeeOne := zigbee.GenerateLocalAdministeredIEEEAddress()
		ieeeTwo := zigbee.GenerateLocalAdministeredIEEEAddress()

		zgw.sectionForNode(ieeeOne)
		zgw.sectionForNode(ieeeTwo)

		assert.True(t, zgw.sectionRemoveNode(ieeeTwo))

		nodes := zgw.nodeListFromPersistence()

		assert.Contains(t, nodes, ieeeOne)
		assert.NotContains(t, nodes, ieeeTwo)
	})
}

func Test_gateway_sectionRemoveDevice(t *testing.T) {
	t.Run("devices is removed", func(t *testing.T) {
		zgw := New(context.Background(), memory.New(), nil, nil)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		deviceOne := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 1}
		deviceTwo := IEEEAddressWithSubIdentifier{IEEEAddress: ieee, SubIdentifier: 2}

		zgw.sectionForDevice(deviceOne)
		zgw.sectionForDevice(deviceTwo)

		assert.True(t, zgw.sectionRemoveDevice(deviceTwo))

		devices := zgw.deviceListFromPersistence(ieee)

		assert.Contains(t, devices, deviceOne)
		assert.NotContains(t, devices, deviceTwo)
	})
}
