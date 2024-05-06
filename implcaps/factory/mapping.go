package factory

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/generic"
	"github.com/shimmeringbee/zda/implcaps/zcl/humidity_sensor"
	"github.com/shimmeringbee/zda/implcaps/zcl/temperature_sensor"
)

const GenericProductInformation = "GenericProductInformation"
const ZCLTemperatureSensor = "ZCLTemperatureSensor"
const ZCLHumiditySensor = "ZCLHumiditySensor"

var Mapping = map[string]da.Capability{
	GenericProductInformation: capabilities.ProductInformationFlag,
	ZCLTemperatureSensor:      capabilities.TemperatureSensorFlag,
	ZCLHumiditySensor:         capabilities.RelativeHumiditySensorFlag,
}

func Create(name string, iface implcaps.ZDAInterface) implcaps.ZDACapability {
	switch name {
	case GenericProductInformation:
		return generic.NewProductInformation()
	case ZCLTemperatureSensor:
		return temperature_sensor.NewTemperatureSensor(iface)
	case ZCLHumiditySensor:
		return humidity_sensor.NewHumiditySensor(iface)
	default:
		return nil
	}
}
