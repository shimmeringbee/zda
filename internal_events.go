package zda

type internalNodeJoin struct {
	node *internalNode
}

type internalNodeLeave struct {
	node *internalNode
}

type internalNodeEnumeration struct {
	node *internalNode
}

type internalDeviceAdded struct {
	device *internalDevice
}

type internalDeviceRemoved struct {
	device *internalDevice
}
