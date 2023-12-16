package factory

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/generic"
)

const GenericProductInformation = "GenericProductInformation"

var Mapping = map[string]da.Capability{
	GenericProductInformation: capabilities.ProductInformationFlag,
}

func Create(name string) implcaps.ZDACapability {
	switch name {
	case GenericProductInformation:
		return generic.NewProductInformation()
	default:
		return nil
	}
}
