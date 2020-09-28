package has_product_information

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
	"sync"
)

type ProductData struct {
	Manufacturer *string
	Product      *string
}

type Implementation struct {
	data     map[zda.IEEEAddressWithSubIdentifier]ProductData
	datalock *sync.RWMutex

	supervisor capability.Supervisor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.HasProductInformationFlag
}

func (i *Implementation) Init(supervisor capability.Supervisor) {
	i.supervisor = supervisor

	i.data = make(map[zda.IEEEAddressWithSubIdentifier]ProductData)
	i.datalock = &sync.RWMutex{}

	i.supervisor.EventSubscription().AddedDevice(i.addedDeviceCallback)
	i.supervisor.EventSubscription().RemovedDevice(i.removedDeviceCallback)
	i.supervisor.EventSubscription().EnumerateDevice(i.enumerateDeviceCallback)
}
