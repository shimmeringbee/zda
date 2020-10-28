package has_product_information

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda"
)

const PersistenceName = "HasProductInformation"

func (i *Implementation) Name() string {
	return PersistenceName
}

func (i *Implementation) DataStruct() interface{} {
	return &ProductData{}
}

func (i *Implementation) Save(d zda.Device) (interface{}, error) {
	if !d.HasCapability(capabilities.HasProductInformationFlag) {
		return nil, da.DeviceDoesNotHaveCapability
	}

	i.datalock.RLock()
	defer i.datalock.RUnlock()

	productData := i.data[d.Identifier]

	return &productData, nil
}

func (i *Implementation) Load(d zda.Device, state interface{}) error {
	if !d.HasCapability(capabilities.HasProductInformationFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	pd, ok := state.(*ProductData)
	if !ok {
		return fmt.Errorf("invalid data structure provided for load")
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	i.data[d.Identifier] = *pd

	return nil
}
