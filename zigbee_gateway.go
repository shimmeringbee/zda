package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
)

type ZigbeeGateway struct {
	provider zigbee.Provider
}

func (z *ZigbeeGateway) Start() error {
	return nil
}

func (z *ZigbeeGateway) Stop() error {
	return nil
}

func (z *ZigbeeGateway) ReadEvent() (interface{}, error) {
	return nil, nil
}

func (z *ZigbeeGateway) Capability(capability da.Capability) interface{} {
	return nil
}

func (z *ZigbeeGateway) Self() da.Device {
	return da.Device{}
}

func (z *ZigbeeGateway) Devices() []da.Device {
	return []da.Device{}
}
