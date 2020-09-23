package zda

import (
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
)

type State struct {
	Nodes map[zigbee.IEEEAddress]StateNode
}

type StateNode struct {
	Devices   map[uint8]StateDevice
	Endpoints []zigbee.EndpointDescription
}

type StateDevice struct {
	DeviceID       uint16
	DeviceVersion  uint8
	Endpoints      []zigbee.Endpoint
	Capabilities   []da.Capability
	CapabilityData map[string]interface{}
}

func (z *ZigbeeGateway) SaveState() State {
	state := State{
		Nodes: map[zigbee.IEEEAddress]StateNode{},
	}

	for _, iNode := range z.nodeTable.getNodes() {
		iNode.mutex.RLock()

		endpointDescriptions := make([]zigbee.EndpointDescription, 0)

		for _, endpointDescription := range iNode.endpointDescriptions {
			endpointDescriptions = append(endpointDescriptions, endpointDescription)
		}

		stateDevices := map[uint8]StateDevice{}

		for subId, iDev := range iNode.devices {
			iDev.mutex.RLock()

			capabilityData := map[string]interface{}{}

			for _, capability := range iDev.capabilities {
				persistingCapability, ok := z.capabilities[capability].(CapabilityPersistentData)
				if ok {
					data, err := persistingCapability.Save(iDev)
					if err == nil {
						capabilityData[persistingCapability.KeyName()] = data
					}
				}
			}

			sDevice := StateDevice{
				DeviceID:       iDev.deviceID,
				DeviceVersion:  iDev.deviceVersion,
				Endpoints:      iDev.endpoints,
				Capabilities:   iDev.capabilities,
				CapabilityData: capabilityData,
			}

			stateDevices[subId] = sDevice

			iDev.mutex.RUnlock()
		}

		sNode := StateNode{
			Devices:   stateDevices,
			Endpoints: endpointDescriptions,
		}

		state.Nodes[iNode.ieeeAddress] = sNode

		iNode.mutex.RUnlock()
	}

	return state
}

func (z *ZigbeeGateway) LoadState(state State) error {
	keyToCapability := map[string]CapabilityPersistentData{}

	for _, capability := range z.capabilities {
		if cpd, ok := capability.(CapabilityPersistentData); ok {
			keyToCapability[cpd.KeyName()] = cpd
		}
	}

	for ieee, stateNode := range state.Nodes {
		iNode, _ := z.nodeTable.createNode(ieee)

		iNode.mutex.Lock()

		for _, ed := range stateNode.Endpoints {
			iNode.endpointDescriptions[ed.Endpoint] = ed
			iNode.endpoints = append(iNode.endpoints, ed.Endpoint)
		}

		iNode.mutex.Unlock()

		for subId, stateDev := range stateNode.Devices {
			iDev, _ := z.nodeTable.createDevice(IEEEAddressWithSubIdentifier{IEEEAddress: iNode.ieeeAddress, SubIdentifier: subId})

			iDev.mutex.Lock()

			iDev.deviceID = stateDev.DeviceID
			iDev.deviceVersion = stateDev.DeviceVersion
			iDev.endpoints = stateDev.Endpoints
			iDev.capabilities = stateDev.Capabilities

			iDev.mutex.Unlock()

			for key, data := range stateDev.CapabilityData {
				capability, found := keyToCapability[key]
				if found {
					if err := capability.Load(iDev, data); err != nil {
						return fmt.Errorf("failed to load data for %s: %w", iDev.generateIdentifier(), err)
					}
				} else {
					return fmt.Errorf("failed to load data for %s: state has unknown capability data", iDev.generateIdentifier())
				}
			}

			z.sendEvent(da.DeviceLoaded{Device: iDev.toDevice(z)})
		}
	}

	return nil
}
