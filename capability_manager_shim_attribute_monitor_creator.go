package zda

import (
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zigbee"
	"sync"
)

type attributeMonitorCreatorShim struct {
	zcl          ZCL
	deviceConfig DeviceConfig
	poller       Poller
}

func (s *attributeMonitorCreatorShim) Create(bc BasicCapability, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, cb func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue)) AttributeMonitor {
	zam := &zclAttributeMonitor{
		zcl:               s.zcl,
		deviceConfig:      s.deviceConfig,
		poller:            s.poller,
		capability:        bc,
		clusterID:         c,
		attributeID:       a,
		attributeDataType: dt,
		callback:          cb,
		deviceListMutex:   &sync.Mutex{},
		deviceList:        map[IEEEAddressWithSubIdentifier]monitorDevice{},
	}

	zam.Init()

	return zam
}
