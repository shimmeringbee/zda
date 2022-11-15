package zda

import (
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_node_nextTransactionSequence(t *testing.T) {
	t.Run("iterates through transaction sequences and wraps at end", func(t *testing.T) {
		n := node{
			sequence: make(chan uint8, 3),
		}

		n.sequence <- 1
		n.sequence <- 2
		n.sequence <- 3

		assert.Equal(t, uint8(1), n.nextTransactionSequence())
		assert.Equal(t, uint8(2), n.nextTransactionSequence())
		assert.Equal(t, uint8(3), n.nextTransactionSequence())
		assert.Equal(t, uint8(1), n.nextTransactionSequence())
	})
}

func Test_node_nextDeviceSubIdentifier(t *testing.T) {
	t.Run("finds the next sub identifier that is available", func(t *testing.T) {
		n := node{
			device: map[uint8]*device{0: nil, 1: nil, 2: nil},
		}

		assert.Equal(t, uint8(3), n._nextDeviceSubIdentifier())
	})
}

func Test_inventory_toRulesInput(t *testing.T) {
	inv := inventory{
		description: &zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		},
		endpoints: map[zigbee.Endpoint]endpointDetails{
			10: {
				description: zigbee.EndpointDescription{
					Endpoint:       zigbee.Endpoint(10),
					ProfileID:      zigbee.ProfileHomeAutomation,
					DeviceID:       0x0400,
					DeviceVersion:  1,
					InClusterList:  []zigbee.ClusterID{0x0000, 0x0006},
					OutClusterList: []zigbee.ClusterID{0x0033},
				},
				productInformation: productData{
					manufacturer: "manufacturer",
					product:      "product",
					version:      "version",
					serial:       "serial",
				},
			},
		},
	}

	ri := rules.Input{
		Node: rules.InputNode{
			ManufacturerCode: 0x1234,
			Type:             "router",
		},
		Self: 0,
		Product: map[int]rules.InputProductData{
			10: {
				Name:         "product",
				Manufacturer: "manufacturer",
				Version:      "version",
				Serial:       "serial",
			},
		},
		Endpoint: map[int]rules.InputEndpoint{
			10: {
				ID:          10,
				ProfileID:   0x0104,
				DeviceID:    0x0400,
				InClusters:  []int{0x0000, 0x0006},
				OutClusters: []int{0x0033},
			},
		},
	}

	assert.Equal(t, ri, inv.toRulesInput())
}
