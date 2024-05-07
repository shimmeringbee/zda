package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_gateway_providerLoad(t *testing.T) {
	t.Run("loads node, device and capability from persistence", func(t *testing.T) {
		s := memory.New()

		g := New(context.Background(), s, nil, nil).(*gateway)
		g.events = make(chan interface{}, 0xffff)

		id := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 1}
		dS := g.sectionForDevice(id)

		cS := dS.Section("capability", "ProductInformation")
		_ = cS.Set("implementation", "GenericProductInformation")

		daS := cS.Section("data")
		_ = daS.Set("Name", "NEXUS-7")
		_ = daS.Set("Manufacturer", "Tyrell Corporation")
		_ = daS.Set("Serial", "N7FAA52318")
		_ = daS.Set("Version", "1.0.0")

		g.providerLoad()

		d := g.getDevice(id)

		c := d.Capability(capabilities.ProductInformationFlag)
		assert.NotNil(t, c)

		cc := c.(capabilities.ProductInformation)
		pi, err := cc.Get(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, "NEXUS-7", pi.Name)
		assert.Equal(t, "Tyrell Corporation", pi.Manufacturer)
		assert.Equal(t, "N7FAA52318", pi.Serial)
		assert.Equal(t, "1.0.0", pi.Version)
	})
}
