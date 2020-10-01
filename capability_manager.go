package zda

import (
	"context"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"time"
)

type CapabilityManager struct {
	gateway                 da.Gateway
	deviceCapabilityManager DeviceCapabilityManager
	eventSender             eventSender
	nodeTable               nodeTable
	callbackAdder           callbacks.Adder
	poller                  poller

	deviceManagerCapability     []DeviceManagementCapability
	deviceEnumerationCapability []DeviceEnumerationCapability

	capabilityByFlag    map[da.Capability]interface{}
	capabilityByKeyName map[string]PersistableCapability
}

func (m *CapabilityManager) Add(c BasicCapability) {
	m.capabilityByFlag[c.Capability()] = c

	if pc, ok := c.(PersistableCapability); ok {
		m.capabilityByKeyName[pc.KeyName()] = pc
	}

	if mc, ok := c.(DeviceManagementCapability); ok {
		m.deviceManagerCapability = append(m.deviceManagerCapability, mc)
	}

	if ec, ok := c.(DeviceEnumerationCapability); ok {
		m.deviceEnumerationCapability = append(m.deviceEnumerationCapability, ec)
	}
}

func (m *CapabilityManager) Get(c da.Capability) interface{} {
	return m.capabilityByFlag[c]
}

func (m *CapabilityManager) PersistingCapabilities() map[string]PersistableCapability {
	return m.capabilityByKeyName
}

func (m *CapabilityManager) Init() {
	supervisor := m.initSupervisor()

	m.callbackAdder.Add(m.deviceAddedCallback)
	m.callbackAdder.Add(m.deviceRemovedCallback)
	m.callbackAdder.Add(m.deviceEnumeratedCallback)

	for _, capability := range m.capabilityByFlag {
		if c, ok := capability.(InitableCapability); ok {
			c.Init(supervisor)
		}
	}
}

func (m *CapabilityManager) deviceAddedCallback(ctx context.Context, e internalDeviceAdded) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceManagerCapability {
		if err := aC.AddedDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}

func (m *CapabilityManager) deviceRemovedCallback(ctx context.Context, e internalDeviceRemoved) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceManagerCapability {
		if err := aC.RemovedDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}

func (m *CapabilityManager) deviceEnumeratedCallback(ctx context.Context, e internalDeviceEnumeration) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceEnumerationCapability {
		if err := aC.EnumerateDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}

func (m *CapabilityManager) Start() {
	for _, capability := range m.capabilityByFlag {
		if c, ok := capability.(ProcessingCapability); ok {
			c.Start()
		}
	}
}

func (m *CapabilityManager) Stop() {
	for _, capability := range m.capabilityByFlag {
		if c, ok := capability.(ProcessingCapability); ok {
			c.Stop()
		}
	}
}

func (m *CapabilityManager) initSupervisor() CapabilitySupervisor {
	return SimpleSupervisor{
		FCImpl:     m,
		MDCImpl:    &manageDeviceCapabilitiesShim{deviceCapabilityManager: m.deviceCapabilityManager},
		CDADImpl:   &composeDADeviceShim{gateway: m.gateway},
		DLImpl:     &deviceLookupShim{nodeTable: m.nodeTable, gateway: m.gateway},
		ZCLImpl:    nil,
		DAESImpl:   &daEventSenderShim{eventSender: m.eventSender},
		PollerImpl: &pollerShim{poller: m.poller},
	}
}

type manageDeviceCapabilitiesShim struct {
	deviceCapabilityManager DeviceCapabilityManager
}

func (s *manageDeviceCapabilitiesShim) Add(d Device, c da.Capability) {
	s.deviceCapabilityManager.AddCapability(d.Identifier, c)
}

func (s *manageDeviceCapabilitiesShim) Remove(d Device, c da.Capability) {
	s.deviceCapabilityManager.RemoveCapability(d.Identifier, c)
}

type daEventSenderShim struct {
	eventSender eventSender
}

func (s *daEventSenderShim) Send(e interface{}) {
	s.eventSender.sendEvent(e)
}

type composeDADeviceShim struct {
	gateway da.Gateway
}

func (s *composeDADeviceShim) Compose(zdaDevice Device) da.Device {
	return da.BaseDevice{
		DeviceGateway:      s.gateway,
		DeviceIdentifier:   zdaDevice.Identifier,
		DeviceCapabilities: zdaDevice.Capabilities,
	}
}

type deviceLookupShim struct {
	gateway   da.Gateway
	nodeTable nodeTable
}

func (s *deviceLookupShim) ByDA(d da.Device) (Device, bool) {
	if s.gateway != d.Gateway() {
		return Device{}, false
	}

	addr, ok := d.Identifier().(IEEEAddressWithSubIdentifier)
	if !ok {
		return Device{}, false
	}

	iDev := s.nodeTable.getDevice(addr)
	if iDev == nil {
		return Device{}, false
	}

	return internalDeviceToZDADevice(iDev), true
}

type pollerShim struct {
	poller poller
}

func (s *pollerShim) Add(d Device, t time.Duration, f func(context.Context, Device) bool) func() {
	isCancelled := false

	s.poller.Add(d.Identifier, t, func(ctx context.Context, iDev *internalDevice) bool {
		if !isCancelled {
			return f(ctx, internalDeviceToZDADevice(iDev))
		} else {
			return false
		}
	})

	return func() {
		isCancelled = true
	}
}

func internalDeviceToZDADevice(iDev *internalDevice) Device {
	iDev.node.mutex.RLock()
	defer iDev.node.mutex.RUnlock()
	iDev.mutex.RLock()
	defer iDev.mutex.RUnlock()

	endpoints := map[zigbee.Endpoint]zigbee.EndpointDescription{}

	for _, endpoint := range iDev.endpoints {
		endpoints[endpoint] = iDev.node.endpointDescriptions[endpoint]
	}

	return Device{
		Identifier: IEEEAddressWithSubIdentifier{
			IEEEAddress:   iDev.node.ieeeAddress,
			SubIdentifier: iDev.subidentifier,
		},
		Capabilities: iDev.capabilities,
		Endpoints:    endpoints,
	}
}
