package generic

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/implcaps"
	"sync"
)

type ProductInformation struct {
	m  *sync.RWMutex
	pi *capabilities.ProductInfo
}

func NewProductInformation() *ProductInformation {
	return &ProductInformation{m: &sync.RWMutex{}}
}

func (g *ProductInformation) ImplName() string {
	return "GenericProductInformation"
}

func (g *ProductInformation) Capability() da.Capability {
	return capabilities.ProductInformationFlag
}

func (g *ProductInformation) Name() string {
	return capabilities.StandardNames[capabilities.ProductInformationFlag]
}

func (g *ProductInformation) Attach(_ context.Context, _ da.Device, _ implcaps.AttachType, m map[string]interface{}) (bool, error) {
	g.m.Lock()
	defer g.m.Unlock()

	newPI := &capabilities.ProductInfo{}

	for k, v := range m {
		stringV, ok := v.(string)
		if !ok {
			return g.pi != nil, fmt.Errorf("failed to cast '%s' value to string", k)
		}

		switch k {
		case "Name":
			newPI.Name = stringV
		case "Manufacturer":
			newPI.Manufacturer = stringV
		case "Version":
			newPI.Version = stringV
		case "Serial":
			newPI.Serial = stringV
		}
	}

	g.pi = newPI
	return true, nil
}

func (g *ProductInformation) Detach(_ context.Context) error {
	return nil
}

func (g *ProductInformation) State() map[string]interface{} {
	g.m.RLock()
	defer g.m.RUnlock()

	return map[string]interface{}{
		"Name":         g.pi.Name,
		"Serial":       g.pi.Serial,
		"Manufacturer": g.pi.Manufacturer,
		"Version":      g.pi.Version,
	}
}

func (g *ProductInformation) Get(_ context.Context) (capabilities.ProductInfo, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	return *g.pi, nil
}

var _ capabilities.ProductInformation = (*ProductInformation)(nil)
var _ implcaps.ZDACapability = (*ProductInformation)(nil)
var _ da.BasicCapability = (*ProductInformation)(nil)
