package zda

import (
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
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
