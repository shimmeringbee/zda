package zda

import (
	. "github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type ZigbeeDevice struct {
	device Device
	mutex  *sync.RWMutex

	nodeDesc             zigbee.NodeDescription
	endpoints            []zigbee.Endpoint
	endpointDescriptions map[zigbee.Endpoint]zigbee.EndpointDescription
}
