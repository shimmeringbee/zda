package factory

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/generic"
	"github.com/shimmeringbee/zda/implcaps/zcl/temperature_sensor"
)

const GenericProductInformation = "GenericProductInformation"
const ZCLTemperatureSensor = "ZCLTemperatureSensor"

var Mapping = map[string]da.Capability{
	GenericProductInformation: capabilities.ProductInformationFlag,
	ZCLTemperatureSensor:      capabilities.TemperatureSensorFlag,
}

func Create(name string, iface implcaps.ZDAInterface) implcaps.ZDACapability {
	switch name {
	case GenericProductInformation:
		return generic.NewProductInformation()
	case ZCLTemperatureSensor:
		return temperature_sensor.NewTemperatureSensor(iface)
	default:
		return nil
	}
}
