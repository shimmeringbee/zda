package zda

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"testing"
)

func TestCapabilityManager_initSupervisor_ManageDeviceCapabilities(t *testing.T) {
	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mdcm := &mockDeviceCapabilityManager{}
		defer mdcm.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{
			Identifier: addr,
		}
		c := da.Capability(0x0001)

		mdcm.On("AddCapability", addr, c)

		m := CapabilityManager{deviceCapabilityManager: mdcm}
		s := m.initSupervisor()

		s.ManageDeviceCapabilities().Add(zdaDevice, c)
	})

	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mdcm := &mockDeviceCapabilityManager{}
		defer mdcm.AssertExpectations(t)

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{
			Identifier: addr,
		}
		c := da.Capability(0x0001)

		mdcm.On("RemoveCapability", addr, c)

		m := CapabilityManager{deviceCapabilityManager: mdcm}
		s := m.initSupervisor()

		s.ManageDeviceCapabilities().Remove(zdaDevice, c)
	})
}
