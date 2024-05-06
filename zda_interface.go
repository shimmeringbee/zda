package zda

import (
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/attribute"
)

type zdaInterface struct {
	gw *gateway
	c  communicator.Communicator
}

func (z zdaInterface) NewAttributeMonitor() attribute.Monitor {
	return attribute.NewMonitor(z.gw.zclCommunicator, z.gw.provider, z.gw.transmissionLookup, z.gw.logger)
}

func (z zdaInterface) SendEvent(a any) {
	z.gw.sendEvent(a)
}
