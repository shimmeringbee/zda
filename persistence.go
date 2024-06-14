package zda

import (
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zigbee"
	"strconv"
)

func (g *gateway) sectionRemoveNode(i zigbee.IEEEAddress) bool {
	return g.section.Section("node").SectionDelete(i.String())
}

func (g *gateway) sectionForNode(i zigbee.IEEEAddress) persistence.Section {
	return g.section.Section("node", i.String())
}

func (g *gateway) nodeListFromPersistence() []zigbee.IEEEAddress {
	var nodeList []zigbee.IEEEAddress

	for _, k := range g.section.Section("node").SectionKeys() {
		if addr, err := strconv.ParseUint(k, 16, 64); err == nil {
			nodeList = append(nodeList, zigbee.IEEEAddress(addr))
		}
	}

	return nodeList
}

func (g *gateway) sectionRemoveDevice(i IEEEAddressWithSubIdentifier) bool {
	return g.sectionForNode(i.IEEEAddress).Section("device").SectionDelete(strconv.Itoa(int(i.SubIdentifier)))
}

func (g *gateway) sectionForDevice(i IEEEAddressWithSubIdentifier) persistence.Section {
	return g.sectionForNode(i.IEEEAddress).Section("device", strconv.Itoa(int(i.SubIdentifier)))
}

func (g *gateway) deviceListFromPersistence(id zigbee.IEEEAddress) []IEEEAddressWithSubIdentifier {
	var deviceList []IEEEAddressWithSubIdentifier

	for _, k := range g.sectionForNode(id).Section("device").SectionKeys() {
		if i, err := strconv.Atoi(k); err == nil {
			deviceList = append(deviceList, IEEEAddressWithSubIdentifier{IEEEAddress: id, SubIdentifier: uint8(i)})
		}
	}

	return deviceList
}
