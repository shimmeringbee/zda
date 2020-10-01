package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCapabilityManager_initSupervisor_CDADevice(t *testing.T) {
	t.Run("provides an implementation that creates a da device from a zda device", func(t *testing.T) {
		zgw := &ZigbeeGateway{}

		m := CapabilityManager{gateway: zgw}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		capabilities := []da.Capability{0x0001}

		zdaDevice := Device{
			Identifier:   addr,
			Capabilities: capabilities,
			Endpoints:    nil,
		}

		d := s.ComposeDADevice().Compose(zdaDevice)

		assert.Equal(t, addr, d.Identifier())
		assert.Equal(t, capabilities, d.Capabilities())
		assert.Equal(t, zgw, d.Gateway())
	})
}
