package zda

import "github.com/shimmeringbee/da"

func NewCapabilityManager() *CapabilityManager {
	return &CapabilityManager{
		capabilityByFlag:    map[da.Capability]interface{}{},
		capabilityByKeyName: map[string]PersistableCapability{},
	}
}

type CapabilityManager struct {
	capabilityByFlag    map[da.Capability]interface{}
	capabilityByKeyName map[string]PersistableCapability
}

func (m *CapabilityManager) Add(c BasicCapability) {
	m.capabilityByFlag[c.Capability()] = c

	if pc, ok := c.(PersistableCapability); ok {
		m.capabilityByKeyName[pc.KeyName()] = pc
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
		FCImpl:     nil,
		MDCImpl:    nil,
		ESImpl:     nil,
		CDADImpl:   nil,
		DLImpl:     nil,
		ZCLImpl:    nil,
		DAESImpl:   nil,
		PollerImpl: nil,
	}
}
