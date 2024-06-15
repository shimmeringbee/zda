package generic

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zda/implcaps"
	"sync"
)

type ProductInformation struct {
	s  persistence.Section
	m  *sync.RWMutex
	pi *capabilities.ProductInfo
}

func NewProductInformation() *ProductInformation {
	return &ProductInformation{m: &sync.RWMutex{}}
}

func (g *ProductInformation) ImplName() string {
	return "GenericProductInformation"
}

func (g *ProductInformation) Init(_ da.Device, section persistence.Section) {
	g.s = section
}

func (g *ProductInformation) Load(_ context.Context) (bool, error) {
	g.m.Lock()
	defer g.m.Unlock()

	g.pi = &capabilities.ProductInfo{}
	g.pi.Name, _ = g.s.String("Name")
	g.pi.Manufacturer, _ = g.s.String("Manufacturer")
	g.pi.Version, _ = g.s.String("Version")
	g.pi.Serial, _ = g.s.String("Serial")

	return true, nil
}

func (g *ProductInformation) Capability() da.Capability {
	return capabilities.ProductInformationFlag
}

func (g *ProductInformation) Name() string {
	return capabilities.StandardNames[capabilities.ProductInformationFlag]
}

func (g *ProductInformation) Enumerate(_ context.Context, m map[string]any) (bool, error) {
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
			g.s.Set("Name", stringV)
		case "Manufacturer":
			newPI.Manufacturer = stringV
			g.s.Set("Manufacturer", stringV)
		case "Version":
			newPI.Version = stringV
			g.s.Set("Version", stringV)
		case "Serial":
			newPI.Serial = stringV
			g.s.Set("Serial", stringV)
		}
	}

	g.pi = newPI
	return true, nil
}

func (g *ProductInformation) Detach(_ context.Context, _ implcaps.DetachType) error {
	return nil
}

func (g *ProductInformation) Get(_ context.Context) (capabilities.ProductInfo, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	return *g.pi, nil
}

var _ capabilities.ProductInformation = (*ProductInformation)(nil)
var _ implcaps.ZDACapability = (*ProductInformation)(nil)
var _ da.BasicCapability = (*ProductInformation)(nil)
