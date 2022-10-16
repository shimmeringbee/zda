package catabilities

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
)

type ZDACapability interface {
	Attach(context.Context, da.Device, da.Gateway, zigbee.Endpoint, map[string]interface{}) (bool, error)
	Detach(context.Context) error
	State(context.Context) (map[string]interface{}, error)
}
