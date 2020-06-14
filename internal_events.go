package zda

type internalNodeJoin struct {
	node *ZigbeeDevice
}

type internalNodeLeave struct {
	node *ZigbeeDevice
}

type internalNodeEnumeration struct {
	node *ZigbeeDevice
}
