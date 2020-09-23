package zda

import "github.com/shimmeringbee/da"

type CapabilityBasic interface {
	Capability() da.Capability
}

type CapabilityStartable interface {
	Start()
}

type CapabilityStopable interface {
	Stop()
}

type CapabilityInitable interface {
	Init()
}

type CapabilityPersistentData interface {
	CapabilityBasic
	KeyName() string
	DataStruct() interface{}
	Save(device *internalDevice) (interface{}, error)
	Load(device *internalDevice, data interface{}) error
}
