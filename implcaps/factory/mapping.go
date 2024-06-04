package factory

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/generic"
	"github.com/shimmeringbee/zda/implcaps/zcl/humidity_sensor"
	"github.com/shimmeringbee/zda/implcaps/zcl/identify"
	"github.com/shimmeringbee/zda/implcaps/zcl/power_supply"
	"github.com/shimmeringbee/zda/implcaps/zcl/pressure_sensor"
	"github.com/shimmeringbee/zda/implcaps/zcl/temperature_sensor"
)

const GenericProductInformation = "GenericProductInformation"
const ZCLTemperatureSensor = "ZCLTemperatureSensor"
const ZCLHumiditySensor = "ZCLHumiditySensor"
const ZCLPressureSensor = "ZCLPressureSensor"
const ZCLIdentify = "ZCLIdentify"
const ZCLPowerSupply = "ZCLPowerSupply"

var Mapping = map[string]da.Capability{
	GenericProductInformation: capabilities.ProductInformationFlag,
	ZCLTemperatureSensor:      capabilities.TemperatureSensorFlag,
	ZCLHumiditySensor:         capabilities.RelativeHumiditySensorFlag,
	ZCLPressureSensor:         capabilities.PressureSensorFlag,
	ZCLIdentify:               capabilities.IdentifyFlag,
	ZCLPowerSupply:            capabilities.PowerSupplyFlag,
}

func Create(name string, iface implcaps.ZDAInterface) implcaps.ZDACapability {
	switch name {
	case GenericProductInformation:
		return generic.NewProductInformation()
	case ZCLTemperatureSensor:
		return temperature_sensor.NewTemperatureSensor(iface)
	case ZCLHumiditySensor:
		return humidity_sensor.NewHumiditySensor(iface)
	case ZCLPressureSensor:
		return pressure_sensor.NewPressureSensor(iface)
	case ZCLIdentify:
		return identify.NewIdentify(iface)
	case ZCLPowerSupply:
		return power_suply.NewPowerSupply(iface)
	default:
		return nil
	}
}
