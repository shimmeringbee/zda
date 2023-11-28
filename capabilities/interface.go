package catabilities

import (
	"context"
	"github.com/shimmeringbee/da"
)

const (
	DataKeyAlreadyInitialised = "Initialised"
	DataKeyZigbeeEndpoint     = "ZigbeeEndpoint"
	DataKeyZigbeeClusterID    = "ZigbeeClusterID"
)

type AttachType int

const (
	Enumeration AttachType = iota
	Load
)

type ZDACapability interface {
	Attach(context.Context, da.Device, AttachType, map[string]interface{}) (bool, error)
	Detach(context.Context) error
	State() map[string]interface{}
}
