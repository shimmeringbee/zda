package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
)

func isCapabilityInSlice(haystack []da.Capability, needle da.Capability) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}

	return false
}

func isClusterIdInSlice(haystack []zigbee.ClusterID, needle zigbee.ClusterID) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}

	return false
}

func isEndpointInSlice(haystack []zigbee.Endpoint, needle zigbee.Endpoint) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}

	return false
}
