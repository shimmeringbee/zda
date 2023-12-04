package generic

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	dacapabilities "github.com/shimmeringbee/da/capabilities"
	capabilities "github.com/shimmeringbee/zda/capabilities"
	"sync"
)

type ProductInformation struct {
	m  *sync.RWMutex
	pi *dacapabilities.ProductInfo
}

func (g *ProductInformation) Attach(_ context.Context, _ da.Device, at capabilities.AttachType, m map[string]interface{}) (bool, error) {
	g.m.Lock()
	defer g.m.Unlock()

	newPI := &dacapabilities.ProductInfo{}

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

func (g *ProductInformation) Get(_ context.Context) (dacapabilities.ProductInfo, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	return *g.pi, nil
}

var _ dacapabilities.ProductInformation = (*ProductInformation)(nil)
var _ capabilities.ZDACapability = (*ProductInformation)(nil)
