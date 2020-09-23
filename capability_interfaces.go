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

type CapabilityPersistentData interface {
	KeyName() string
	DataStruct() interface{}
	Save(device *internalDevice) (interface{}, error)
	Load(device *internalDevice, data interface{}) error
}
