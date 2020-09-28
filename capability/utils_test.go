package capability

import (
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindEndpointsWithClusterID(t *testing.T) {
	t.Run("endpoints with the specified cluster are returned", func(t *testing.T) {
		desiredCluster := zigbee.ClusterID(0x7f00)

		d := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{
				0x01: {
					Endpoint:      0x01,
					InClusterList: []zigbee.ClusterID{desiredCluster},
				},
				0x02: {
					Endpoint:      0x02,
					InClusterList: []zigbee.ClusterID{},
				},
				0x03: {
					Endpoint:      0x03,
					InClusterList: []zigbee.ClusterID{desiredCluster},
				},
			},
		}

		expectedEndpoints := []zigbee.Endpoint{0x01, 0x03}
		actualEndpoints := FindEndpointsWithClusterID(d, desiredCluster)

		assert.Equal(t, expectedEndpoints, actualEndpoints)
	})
}
