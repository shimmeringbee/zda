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

func addCapability(device *da.Device, capability da.Capability) {
	if !device.HasCapability(capability) {
		device.Capabilities = append(device.Capabilities, capability)
	}
}

func removeCapability(device *da.Device, capability da.Capability) {
	var newCapabilities []da.Capability

	for _, existingCapability := range device.Capabilities {
		if existingCapability != capability {
			newCapabilities = append(newCapabilities, existingCapability)
		}
	}

	device.Capabilities = newCapabilities
}

func findEndpointWithClusterId(node *internalNode, device *internalDevice, clusterId zigbee.ClusterID) (zigbee.Endpoint, bool) {
	for _, endpoint := range device.endpoints {
		if isClusterIdInSlice(node.endpointDescriptions[endpoint].InClusterList, clusterId) {
			return endpoint, true
		}
	}

	return 0, false
}
