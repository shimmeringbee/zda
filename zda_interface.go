package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zda/attribute"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zigbee"
)

var _ implcaps.ZDAInterface = (*zdaInterface)(nil)

type zdaInterface struct {
	gw *gateway
	c  communicator.Communicator
}

func (z zdaInterface) Logger() logwrap.Logger {
	return z.gw.logger
}

func (z zdaInterface) ZCLRegister(f func(*zcl.CommandRegistry)) {
	f(z.gw.zclCommandRegistry)
}

func (z zdaInterface) TransmissionLookup(d da.Device, id zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8) {
	return z.gw.transmissionLookup(d, id)
}

func (z zdaInterface) ZCLCommunicator() communicator.Communicator {
	return z.c
}

func (z zdaInterface) NewAttributeMonitor() attribute.Monitor {
	return attribute.NewMonitor(z.gw.zclCommunicator, z.gw.provider, z.gw.transmissionLookup, z.gw.logger)
}

func (z zdaInterface) SendEvent(a any) {
	z.gw.sendEvent(a)
}
