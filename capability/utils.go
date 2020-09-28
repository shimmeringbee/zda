package capability

import "github.com/shimmeringbee/zigbee"

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
