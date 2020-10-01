package has_product_information

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
	"sync"
)

type ProductData struct {
	Manufacturer *string
	Product      *string
}

type Implementation struct {
	data     map[zda.IEEEAddressWithSubIdentifier]ProductData
	datalock *sync.RWMutex

	supervisor zda.CapabilitySupervisor
}

func (i *Implementation) Capability() da.Capability {
	return capabilities.HasProductInformationFlag
}

func (i *Implementation) Init(supervisor zda.CapabilitySupervisor) {
	i.supervisor = supervisor

	i.data = make(map[zda.IEEEAddressWithSubIdentifier]ProductData)
	i.datalock = &sync.RWMutex{}
}
