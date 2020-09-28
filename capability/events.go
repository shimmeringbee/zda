package capability

type AddedDevice struct {
	Device Device
}

type RemovedDevice struct {
	Device Device
}

type EnumerateDevice struct {
	Device Device
}

type PollDevice struct {
	Device Device
}