package zda

import "github.com/shimmeringbee/da"

type device struct {
}

func (d device) Gateway() da.Gateway {
	//TODO implement me
	panic("implement me")
}

func (d device) Identifier() da.Identifier {
	//TODO implement me
	panic("implement me")
}

func (d device) Capabilities() []da.Capability {
	//TODO implement me
	panic("implement me")
}

func (d device) HasCapability(c da.Capability) bool {
	//TODO implement me
	panic("implement me")
}

var _ da.Device = (*device)(nil)