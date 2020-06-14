package zda

type CapabilityStartable interface {
	Start()
}

type CapabilityStopable interface {
	Stop()
}

type CapabilityInitable interface {
	Init()
}
