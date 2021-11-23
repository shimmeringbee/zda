package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
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

	capability        da.BasicCapability
	clusterID         zigbee.ClusterID
	attributeID       zcl.AttributeID
	attributeDataType zcl.AttributeDataType
	callback          func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue)

	deviceListMutex *sync.Mutex
	deviceList      map[IEEEAddressWithSubIdentifier]monitorDevice
	logger          logwrap.Logger
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
	cfg := z.deviceConfig.Get(d, z.capability.Name())

	attemptBinding := cfg.Bool("AttemptBinding", true)
	attemptReporting := cfg.Bool("AttemptReporting", true)

	requiresPolling := false

	if attemptBinding {
		err := z.zcl.Bind(ctx, d, e, z.clusterID)
		if err != nil {
			z.logger.LogWarn(ctx, "Failed to bind cluster for attribute monitor.", logwrap.Datum("endpoint", e), logwrap.Datum("clusterID", z.clusterID), logwrap.Err(err))
			requiresPolling = true
		}
	}

	if attemptReporting {
		minimumReportingInterval := cfg.Int("MinimumReportingInterval", 60)
		maximumReportingInterval := cfg.Int("MaximumReportingInterval", 300)

		err := z.zcl.ConfigureReporting(ctx, d, e, z.clusterID, z.attributeID, z.attributeDataType, uint16(minimumReportingInterval), uint16(maximumReportingInterval), v)
		if err != nil {
			z.logger.LogWarn(ctx, "Failed to configure reporting for attribute monitor.", logwrap.Datum("endpoint", e), logwrap.Datum("clusterID", z.clusterID), logwrap.Datum("attributeID", z.attributeID), logwrap.Err(err))
			requiresPolling = true
		}
	}

	if requiresPolling {
		z.Reattach(ctx, d, e, true)
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

func (z *zclAttributeMonitor) Reattach(ctx context.Context, d Device, e zigbee.Endpoint, requiresPolling bool) {
	z.deviceListMutex.Lock()
	existing := z.deviceList[d.Identifier]
	existing.endpoint = e

	if requiresPolling {
		cfg := z.deviceConfig.Get(d, z.capability.Name())

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

	z.zcl.ReadAttributes(ctx, d, deviceData.endpoint, z.clusterID, []zcl.AttributeID{z.attributeID})

	return true
}

func (z *zclAttributeMonitor) zclFilter(_ zigbee.IEEEAddress, _ zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
	_, canCastToReport := zclMessage.Command.(*global.ReportAttributes)
	_, canCastToRead := zclMessage.Command.(*global.ReadAttributesResponse)

	return zclMessage.ClusterID == z.clusterID && (canCastToReport || canCastToRead)
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
	case *global.ReadAttributesResponse:
		for _, record := range cmd.Records {
			if record.Identifier == z.attributeID && record.Status == 0 && record.DataTypeValue.DataType == z.attributeDataType {
				z.callback(d, record.Identifier, *record.DataTypeValue)
			}
		}
	}
}
