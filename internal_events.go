package zda

import "github.com/shimmeringbee/zigbee"

type internalNodeJoin struct {
	node *ZigbeeDevice
}

type internalNodeLeave struct {
	node *ZigbeeDevice
}

type internalNodeEnumeration struct {
	node *ZigbeeDevice
}

type internalNodeIncomingMessage struct {
	node    *ZigbeeDevice
	message zigbee.NodeIncomingMessageEvent
}
