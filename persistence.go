package zda

import (
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"sort"
)

type State struct {
	Nodes map[zigbee.IEEEAddress]StateNode
}

type StateNode struct {
	Devices        map[uint8]StateDevice
	Endpoints      []zigbee.EndpointDescription
	Description    zigbee.NodeDescription
	SupportsAPSAck bool
}

type StateDevice struct {
	DeviceID       uint16
	DeviceVersion  uint8
	Endpoints      []zigbee.Endpoint
	Capabilities   []da.Capability
	CapabilityData map[string]interface{}
}

func internalDeviceToCapabilityDevice(iDev *internalDevice) Device {
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

		sort.Slice(endpointDescriptions, func(i, j int) bool {
			return endpointDescriptions[i].Endpoint < endpointDescriptions[j].Endpoint
		})

		stateDevices := map[uint8]StateDevice{}

		for subId, iDev := range iNode.devices {
			iDev.mutex.RLock()

			capabilityData := map[string]interface{}{}

			for _, capability := range iDev.capabilities {
				persistingCapability, ok := z.CapabilityManager.Get(capability).(PersistableCapability)
				if ok {
					data, err := persistingCapability.Save(internalDeviceToCapabilityDevice(iDev))
					if err == nil {
						capabilityData[persistingCapability.Name()] = data
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
			Devices:        stateDevices,
			Endpoints:      endpointDescriptions,
			SupportsAPSAck: iNode.supportsAPSAck,
			Description:    iNode.nodeDesc,
		}

		state.Nodes[iNode.ieeeAddress] = sNode

		iNode.mutex.RUnlock()
	}

	return state
}

func (z *ZigbeeGateway) LoadState(state State) error {
	keyToCapability := z.CapabilityManager.PersistingCapabilities()

	for ieee, stateNode := range state.Nodes {
		iNode, _ := z.nodeTable.createNode(ieee)

		iNode.mutex.Lock()

		for _, ed := range stateNode.Endpoints {
			iNode.endpointDescriptions[ed.Endpoint] = ed
			iNode.endpoints = append(iNode.endpoints, ed.Endpoint)
		}

		iNode.nodeDesc = stateNode.Description
		iNode.supportsAPSAck = stateNode.SupportsAPSAck

		iNode.mutex.Unlock()

		for subId, stateDev := range stateNode.Devices {
			iDev, _ := z.nodeTable.createDevice(IEEEAddressWithSubIdentifier{IEEEAddress: iNode.ieeeAddress, SubIdentifier: subId})

			iDev.mutex.Lock()

			iDev.deviceID = stateDev.DeviceID
			iDev.deviceVersion = stateDev.DeviceVersion
			iDev.endpoints = stateDev.Endpoints
			iDev.capabilities = stateDev.Capabilities

			if !isCapabilityInSlice(iDev.capabilities, capabilities.EnumerateDeviceFlag) {
				iDev.capabilities = append(iDev.capabilities, capabilities.EnumerateDeviceFlag)
			}

			if !isCapabilityInSlice(iDev.capabilities, capabilities.DeviceRemovalFlag) {
				iDev.capabilities = append(iDev.capabilities, capabilities.DeviceRemovalFlag)
			}

			iDev.mutex.Unlock()

			for key, data := range stateDev.CapabilityData {
				capability, found := keyToCapability[key]
				if found {
					if err := capability.Load(internalDeviceToCapabilityDevice(iDev), data); err != nil {
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

func JSONMarshalState(state State) ([]byte, error) {
	return json.Marshal(state)
}

func JSONUnmarshalState(z *ZigbeeGateway, data []byte) (State, error) {
	state := &State{}

	if err := json.Unmarshal(data, state); err != nil {
		return *state, fmt.Errorf("failed to unmarshal state, stage 1: %w", err)
	}

	keyToCapability := z.CapabilityManager.PersistingCapabilities()

	for _, node := range state.Nodes {
		for _, device := range node.Devices {
			for key, anonymousData := range device.CapabilityData {
				capability, found := keyToCapability[key]
				if !found {
					return *state, fmt.Errorf("failed to find capability to unmarshal: %s", key)
				}

				data, err := json.Marshal(anonymousData)
				if err != nil {
					return *state, fmt.Errorf("failed to find remarshal capability data for key %s: %w", key, err)
				}

				capabilityState := capability.DataStruct()

				if err := json.Unmarshal(data, capabilityState); err != nil {
					return *state, fmt.Errorf("failed to find unmarshal capability data into data struct for key %s: %w", key, err)
				}

				device.CapabilityData[key] = capabilityState
			}
		}
	}

	return *state, nil
}
