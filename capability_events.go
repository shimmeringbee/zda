package zda

type AddedDeviceEvent struct {
	Device Device
}

type RemovedDeviceEvent struct {
	Device Device
}

type EnumerateDeviceEvent struct {
	Device Device
}
