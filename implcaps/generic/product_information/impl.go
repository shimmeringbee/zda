package product_information

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zda/implcaps"
	"sync"
)

type Implementation struct {
	s  persistence.Section
	m  *sync.RWMutex
	pi *capabilities.ProductInfo
}

func NewProductInformation() *Implementation {
	return &Implementation{m: &sync.RWMutex{}}
}

func (g *Implementation) ImplName() string {
	return "GenericProductInformation"
}

func (g *Implementation) Init(_ da.Device, section persistence.Section) {
	g.s = section
}

func (g *Implementation) Load(_ context.Context) (bool, error) {
	g.m.Lock()
	defer g.m.Unlock()

	g.pi = &capabilities.ProductInfo{}
	g.pi.Name, _ = g.s.String("Name")
	g.pi.Manufacturer, _ = g.s.String("Manufacturer")
	g.pi.Version, _ = g.s.String("Version")
	g.pi.Serial, _ = g.s.String("Serial")

	return true, nil
}

func (g *Implementation) Capability() da.Capability {
	return capabilities.ProductInformationFlag
}

func (g *Implementation) Name() string {
	return capabilities.StandardNames[capabilities.ProductInformationFlag]
}

func (g *Implementation) Enumerate(_ context.Context, m map[string]any) (bool, error) {
	g.m.Lock()
	defer g.m.Unlock()

	newPI := &capabilities.ProductInfo{}
	attach := false

	for k, v := range m {
		stringV, ok := v.(string)
		if !ok {
			return g.pi != nil, fmt.Errorf("failed to cast '%s' value to string", k)
		}

		if len(stringV) > 0 {
			switch k {
			case "Name":
				newPI.Name = stringV
				g.s.Set("Name", stringV)
				attach = true
			case "Manufacturer":
				newPI.Manufacturer = stringV
				g.s.Set("Manufacturer", stringV)
				attach = true
			case "Version":
				newPI.Version = stringV
				g.s.Set("Version", stringV)
				attach = true
			case "Serial":
				newPI.Serial = stringV
				g.s.Set("Serial", stringV)
				attach = true
			}
		}
	}

	if attach {
		g.pi = newPI
	}

	return attach, nil
}

func (g *Implementation) Detach(_ context.Context, _ implcaps.DetachType) error {
	return nil
}

func (g *Implementation) Get(_ context.Context) (capabilities.ProductInfo, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	return *g.pi, nil
}

var _ capabilities.ProductInformation = (*Implementation)(nil)
var _ implcaps.ZDACapability = (*Implementation)(nil)
var _ da.BasicCapability = (*Implementation)(nil)
