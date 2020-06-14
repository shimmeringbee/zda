package zda

import (
	"github.com/shimmeringbee/zcl"
)

type internalNodeJoin struct {
	node *ZigbeeDevice
}

type internalNodeLeave struct {
	node *ZigbeeDevice
}

type internalNodeEnumeration struct {
	node *ZigbeeDevice
}

type internalZCLMessage struct {
	node    *ZigbeeDevice
	message zcl.Message
}
