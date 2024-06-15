package zda

import (
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zigbee"
	"strconv"
)

func (z *ZDA) sectionRemoveNode(i zigbee.IEEEAddress) bool {
	return z.section.Section("node").SectionDelete(i.String())
}

func (z *ZDA) sectionForNode(i zigbee.IEEEAddress) persistence.Section {
	return z.section.Section("node", i.String())
}

func (z *ZDA) nodeListFromPersistence() []zigbee.IEEEAddress {
	var nodeList []zigbee.IEEEAddress

	for _, k := range z.section.Section("node").SectionKeys() {
		if addr, err := strconv.ParseUint(k, 16, 64); err == nil {
			nodeList = append(nodeList, zigbee.IEEEAddress(addr))
		}
	}

	return nodeList
}

func (z *ZDA) sectionRemoveDevice(i IEEEAddressWithSubIdentifier) bool {
	return z.sectionForNode(i.IEEEAddress).Section("device").SectionDelete(strconv.Itoa(int(i.SubIdentifier)))
}

func (z *ZDA) sectionForDevice(i IEEEAddressWithSubIdentifier) persistence.Section {
	return z.sectionForNode(i.IEEEAddress).Section("device", strconv.Itoa(int(i.SubIdentifier)))
}

func (z *ZDA) deviceListFromPersistence(id zigbee.IEEEAddress) []IEEEAddressWithSubIdentifier {
	var deviceList []IEEEAddressWithSubIdentifier

	for _, k := range z.sectionForNode(id).Section("device").SectionKeys() {
		if i, err := strconv.Atoi(k); err == nil {
			deviceList = append(deviceList, IEEEAddressWithSubIdentifier{IEEEAddress: id, SubIdentifier: uint8(i)})
		}
	}

	return deviceList
}
