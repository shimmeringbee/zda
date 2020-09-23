package zda

import (
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
	DeviceID      uint16
	DeviceVersion uint8
	Endpoints     []zigbee.Endpoint
	Capabilities  []da.Capability
}

func (z *ZigbeeGateway) SaveState() State {
	state := State{
		Nodes: map[zigbee.IEEEAddress]StateNode{},
	}

	for _, iNode := range z.nodeTable.getNodes() {
		iNode.mutex.RLock()

		var endpointDescriptions []zigbee.EndpointDescription

		for _, endpointDescription := range iNode.endpointDescriptions {
			endpointDescriptions = append(endpointDescriptions, endpointDescription)
		}

		stateDevices := map[uint8]StateDevice{}

		for subId, iDev := range iNode.devices {
			iDev.mutex.RLock()

			sDevice := StateDevice{
				DeviceID:      iDev.deviceID,
				DeviceVersion: iDev.deviceVersion,
				Endpoints:     iDev.endpoints,
				Capabilities:  iDev.capabilities,
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

func (z *ZigbeeGateway) LoadState(state State) {
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

			z.sendEvent(da.DeviceLoaded{Device: iDev.toDevice(z)})
		}
	}
}
