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

func isUint8InSlice(haystack []uint8, needle uint8) bool {
	for _, piece := range haystack {
		if piece == needle {
			return true
		}
	}

	return false
}

func FindEndpointsWithClusterID(device Device, clusterId zigbee.ClusterID) []zigbee.Endpoint {
	var endpoints []zigbee.Endpoint

	for _, endpointDescription := range device.Endpoints {

		if IsClusterIdInSlice(endpointDescription.InClusterList, clusterId) {
			endpoints = append(endpoints, endpointDescription.Endpoint)
		}
	}

	return endpoints
}

func IsClusterIdInSlice(haystack []zigbee.ClusterID, needle zigbee.ClusterID) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}

	return false
}
