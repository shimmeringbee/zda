package zda

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"sync"
	"time"
)

type zclAttributeMonitor struct {
	zcl          ZCL
	deviceConfig DeviceConfig
	poller       Poller

	capability        BasicCapability
	clusterID         zigbee.ClusterID
	attributeID       zcl.AttributeID
	attributeDataType zcl.AttributeDataType
	callback          func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue)

	deviceListMutex *sync.Mutex
	deviceList      map[IEEEAddressWithSubIdentifier]monitorDevice
}

type monitorDevice struct {
	pollerCancel func()
	endpoint     zigbee.Endpoint
}

const DefaultPollingInterval = 5 * time.Second

func (z *zclAttributeMonitor) Init() {
	z.zcl.Listen(z.zclFilter, z.zclMessage)
}

func (z *zclAttributeMonitor) Attach(ctx context.Context, d Device, e zigbee.Endpoint, v interface{}) (bool, error) {
	cfg := z.deviceConfig.Get(d, z.capability.KeyName())

	attemptBinding := cfg.Bool("AttemptBinding", true)
	attemptReporting := cfg.Bool("AttemptReporting", true)

	requiresPolling := false

	if attemptBinding {
		err := z.zcl.Bind(ctx, d, e, z.clusterID)
		if err != nil {
			requiresPolling = true
		}
	}

	if attemptReporting {
		minimumReportingInterval := cfg.Int("MinimumReportingInterval", 0)
		maximumReportingInterval := cfg.Int("MaximumReportingInterval", 60)

		err := z.zcl.ConfigureReporting(ctx, d, e, z.clusterID, z.attributeID, z.attributeDataType, uint16(minimumReportingInterval), uint16(maximumReportingInterval), v)
		if err != nil {
			requiresPolling = true
		}
	}

	if requiresPolling {
		z.Load(ctx, d, e, true)
	} else {
		z.deviceListMutex.Lock()
		z.deviceList[d.Identifier] = monitorDevice{endpoint: e}
		z.deviceListMutex.Unlock()
	}

	return requiresPolling, nil
}

func (z *zclAttributeMonitor) Detach(ctx context.Context, d Device) {
	z.deviceListMutex.Lock()
	defer z.deviceListMutex.Unlock()

	if existing, found := z.deviceList[d.Identifier]; found {
		if existing.pollerCancel != nil {
			existing.pollerCancel()
		}

		delete(z.deviceList, d.Identifier)
	}
}

func (z *zclAttributeMonitor) Load(ctx context.Context, d Device, e zigbee.Endpoint, requiresPolling bool) {
	z.deviceListMutex.Lock()
	existing := z.deviceList[d.Identifier]
	existing.endpoint = e

	if requiresPolling {
		cfg := z.deviceConfig.Get(d, z.capability.KeyName())

		if existing.pollerCancel != nil {
			existing.pollerCancel()
		}

		pollingInterval := cfg.Duration("PollingInterval", DefaultPollingInterval)
		existing.pollerCancel = z.poller.Add(d, pollingInterval, z.actualPollDevice)
	}

	z.deviceList[d.Identifier] = existing
	z.deviceListMutex.Unlock()
}

func (z *zclAttributeMonitor) Poll(ctx context.Context, d Device) {
	z.actualPollDevice(ctx, d)
}

func (z *zclAttributeMonitor) actualPollDevice(ctx context.Context, d Device) bool {
	z.deviceListMutex.Lock()
	deviceData, found := z.deviceList[d.Identifier]
	z.deviceListMutex.Unlock()

	if !found {
		return false
	}

	results, err := z.zcl.ReadAttributes(ctx, d, deviceData.endpoint, z.clusterID, []zcl.AttributeID{z.attributeID})
	if err == nil {
		wantedAttribute, found := results[z.attributeID]

		if found && wantedAttribute.Status == 0 {
			z.callback(d, z.attributeID, *wantedAttribute.DataTypeValue)
		}
	}

	return true
}

func (z *zclAttributeMonitor) zclFilter(_ zigbee.IEEEAddress, _ zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
	_, canCast := zclMessage.Command.(*global.ReportAttributes)
	return zclMessage.ClusterID == z.clusterID && canCast
}

func (z *zclAttributeMonitor) zclMessage(d Device, m zcl.Message) {
	if !d.HasCapability(z.capability.Capability()) {
		return
	}

	switch cmd := m.Command.(type) {
	case *global.ReportAttributes:
		for _, record := range cmd.Records {
			if record.Identifier == z.attributeID && record.DataTypeValue.DataType == z.attributeDataType {
				z.callback(d, record.Identifier, *record.DataTypeValue)
			}
		}
	}
}
