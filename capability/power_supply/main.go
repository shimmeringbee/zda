package power_supply

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/power_configuration"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type Data struct {
	Mains                   []*capabilities.PowerMainsStatus
	Battery                 []*capabilities.PowerBatteryStatus
	RequiresPolling         bool
	Endpoint                zigbee.Endpoint
	PowerConfiguration      bool
	VendorXiaomiApproachOne bool
}

type PersistentData struct {
	Mains                   []capabilities.PowerMainsStatus
	Battery                 []capabilities.PowerBatteryStatus
	RequiresPolling         bool
	Endpoint                zigbee.Endpoint
	PowerConfiguration      bool
	VendorXiaomiApproachOne bool
}

type Implementation struct {
	supervisor zda.CapabilitySupervisor

	data     map[zda.IEEEAddressWithSubIdentifier]Data
	datalock *sync.RWMutex

	attMonMainsVoltage               zda.AttributeMonitor
	attMonMainsFrequency             zda.AttributeMonitor
	attMonBatteryVoltage             zda.AttributeMonitor
	attMonBatteryPercentageRemaining zda.AttributeMonitor
	attMonVendorXiaomiApproachOne    zda.AttributeMonitor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.PowerSupplyFlag
}

func (i *Implementation) Name() string {
	return capabilities.StandardNames[i.Capability()]
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = map[zda.IEEEAddressWithSubIdentifier]Data{}
	i.datalock = &sync.RWMutex{}

	i.attMonMainsVoltage = i.supervisor.AttributeMonitorCreator().Create(i, zcl.PowerConfigurationId, power_configuration.MainsVoltage, zcl.TypeUnsignedInt16, i.attributeUpdateMainsVoltage)
	i.attMonMainsFrequency = i.supervisor.AttributeMonitorCreator().Create(i, zcl.PowerConfigurationId, power_configuration.MainsFrequency, zcl.TypeUnsignedInt8, i.attributeUpdateMainsFrequency)
	i.attMonBatteryVoltage = i.supervisor.AttributeMonitorCreator().Create(i, zcl.PowerConfigurationId, power_configuration.BatteryVoltage, zcl.TypeUnsignedInt8, i.attributeUpdateBatteryVoltage)
	i.attMonBatteryPercentageRemaining = i.supervisor.AttributeMonitorCreator().Create(i, zcl.PowerConfigurationId, power_configuration.BatteryPercentageRemaining, zcl.TypeUnsignedInt8, i.attributeUpdateBatterPercentageRemaining)
	i.attMonVendorXiaomiApproachOne = i.supervisor.AttributeMonitorCreator().Create(i, zcl.BasicId, zcl.AttributeID(0xff01), zcl.TypeStringCharacter8, i.attributeUpdateVendorXiaomiApproachOne)
}
