package attribute

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/persistence/converter"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"math"
	"time"
)

type PollingMode int

const (
	PollIfReportingFailed PollingMode = iota
	AlwaysPoll
	NeverPoll
)

type PollingConfig struct {
	Mode     PollingMode
	Interval time.Duration
}

type ReportingMode int

const (
	AttemptConfigureReporting ReportingMode = iota
	NeverConfigureReporting
)

type ReportingConfig struct {
	Mode             ReportingMode
	MinimumInterval  time.Duration
	MaximumInterval  time.Duration
	ReportableChange any
}

type MonitorCallback func(zcl.AttributeID, zcl.AttributeDataTypeValue)

type Monitor interface {
	Init(s persistence.Section, d da.Device, cb MonitorCallback)
	Load(ctx context.Context) error
	Attach(ctx context.Context, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, rc ReportingConfig, pc PollingConfig) error
	Detach(ctx context.Context, unconfigure bool) error
}

const ReportingConfiguredKey = "ReportingConfigured"
const PollingConfiguredKey = "PollingConfigured"
const PollingIntervalKey = "PollingInterval"

const RemoteEndpointKey = "RemoteEndpoint"
const ClusterIdKey = "ClusterID"
const AttributeIdKey = "AttributeID"
const AttributeDataTypeKey = "AttributeDataType"

type transmissionLookup func(da.Device, zigbee.ProfileID) (zigbee.IEEEAddress, zigbee.Endpoint, bool, uint8)

func NewMonitor(c communicator.Communicator, nb zigbee.NodeBinder, tl transmissionLookup, l logwrap.Logger) Monitor {
	return &zclMonitor{
		zclCommunicator:    c,
		nodeBinder:         nb,
		transmissionLookup: tl,
		logger:             &l,
		pollerStop:         make(chan struct{}, 1),
	}
}

type zclMonitor struct {
	zclCommunicator    communicator.Communicator
	nodeBinder         zigbee.NodeBinder
	transmissionLookup transmissionLookup
	logger             *logwrap.Logger

	config   persistence.Section
	device   da.Device
	callback MonitorCallback
	match    communicator.Match

	ieeeAddress       zigbee.IEEEAddress
	remoteEndpoint    zigbee.Endpoint
	localEndpoint     zigbee.Endpoint
	clusterID         zigbee.ClusterID
	attributeID       zcl.AttributeID
	attributeDataType zcl.AttributeDataType

	ticker     *time.Ticker
	pollerStop chan struct{}
}

func (z *zclMonitor) Init(s persistence.Section, d da.Device, cb MonitorCallback) {
	z.config = s
	z.device = d
	z.callback = cb

	z.logger.AddOptionsToLogger(logwrap.Datum("Identifier", d.Identifier().String()))
}

func (z *zclMonitor) Load(pctx context.Context) error {
	ctx, end := z.logger.Segment(pctx, "Loading attribute monitor.")
	defer end()

	z.ieeeAddress, z.localEndpoint, _, _ = z.transmissionLookup(z.device, zigbee.ProfileHomeAutomation)

	if v, ok := z.config.Int(RemoteEndpointKey); ok {
		z.remoteEndpoint = zigbee.Endpoint(v)
	} else {
		z.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", RemoteEndpointKey))
		return fmt.Errorf("monitor missing config parameter: %s", RemoteEndpointKey)
	}

	if v, ok := z.config.Int(ClusterIdKey); ok {
		z.clusterID = zigbee.ClusterID(v)
	} else {
		z.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", ClusterIdKey))
		return fmt.Errorf("monitor missing config parameter: %s", ClusterIdKey)
	}

	if v, ok := z.config.Int(AttributeIdKey); ok {
		z.attributeID = zcl.AttributeID(v)
	} else {
		z.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", AttributeIdKey))
		return fmt.Errorf("monitor missing config parameter: %s", AttributeIdKey)
	}

	if v, ok := z.config.Int(AttributeDataTypeKey); ok {
		z.attributeDataType = zcl.AttributeDataType(v)
	} else {
		z.logger.Error(ctx, "Required config parameter missing.", logwrap.Datum("name", AttributeDataTypeKey))
		return fmt.Errorf("monitor missing config parameter: %s", AttributeDataTypeKey)
	}

	return z.reattach(ctx)
}

func (z *zclMonitor) reattach(ctx context.Context) error {
	z.match = communicator.NewMatch(z.zclFilter, z.zclMessage)
	z.zclCommunicator.RegisterMatch(z.match)

	z.logger.Info(ctx, "Attribute monitor configuration.", logwrap.Data(logwrap.List{"LocalEndpoint": z.localEndpoint, "RemoteEndpoint": z.remoteEndpoint, "ClusterId": z.clusterID, "AttributeID": z.attributeID, "AttributeType": z.attributeDataType}))

	// If polling, start timer.
	if v, ok := z.config.Bool(PollingConfiguredKey); ok && v {
		interval, _ := converter.Retrieve(z.config, PollingIntervalKey, converter.DurationDecoder, time.Duration(5)*time.Minute)
		duration := time.Duration(interval) * time.Millisecond

		z.logger.Info(ctx, "Polling configured, starting...", logwrap.Datum("intervalMs", duration.Milliseconds()))

		z.ticker = time.NewTicker(duration)
		go z.poller(context.TODO())
	}

	return nil
}

func (z *zclMonitor) Attach(ctx context.Context, e zigbee.Endpoint, c zigbee.ClusterID, a zcl.AttributeID, dt zcl.AttributeDataType, rc ReportingConfig, pc PollingConfig) error {
	var ack bool
	var seq uint8

	z.logger.Info(ctx, "Attaching event monitor...", logwrap.Datum("ReportingMode", rc.Mode), logwrap.Datum("PollingMode", rc.Mode))

	z.ieeeAddress, z.localEndpoint, ack, seq = z.transmissionLookup(z.device, zigbee.ProfileHomeAutomation)

	var failedReporting = false

	z.remoteEndpoint = e
	z.clusterID = c
	z.attributeID = a
	z.attributeDataType = dt

	z.config.Set(RemoteEndpointKey, int(z.remoteEndpoint))
	z.config.Set(ClusterIdKey, int(z.clusterID))
	z.config.Set(AttributeIdKey, int(z.attributeID))
	z.config.Set(AttributeDataTypeKey, int(z.attributeDataType))

	if rc.Mode == AttemptConfigureReporting {
		z.logger.Info(ctx, "Attempting to configure attribute reporting.")

		if err := z.nodeBinder.BindNodeToController(ctx, z.ieeeAddress, z.localEndpoint, e, c); err != nil {
			z.logger.Warn(ctx, "Binding node to controller failed.", logwrap.Err(err))
			failedReporting = true
		} else {
			if err := z.zclCommunicator.ConfigureReporting(ctx, z.ieeeAddress, ack, z.clusterID, zigbee.NoManufacturer, z.localEndpoint, z.remoteEndpoint, seq, z.attributeID, z.attributeDataType, uint16(math.Round(rc.MinimumInterval.Seconds())), uint16(math.Round(rc.MaximumInterval.Seconds())), rc.ReportableChange); err != nil {
				z.logger.Warn(ctx, "Configure reporting failed.", logwrap.Err(err))
				failedReporting = true
			} else {
				z.config.Set(ReportingConfiguredKey, true)
				z.logger.Info(ctx, "Reporting configured successfully.")
			}
		}
	}

	if (failedReporting && pc.Mode == PollIfReportingFailed) || pc.Mode == AlwaysPoll {
		z.config.Set(PollingConfiguredKey, true)
		converter.Store(z.config, PollingIntervalKey, pc.Interval, converter.DurationEncoder)
	}

	return z.reattach(ctx)
}

func (z *zclMonitor) Detach(ctx context.Context, unconfigure bool) error {
	z.logger.Info(ctx, "Detaching event monitor...", logwrap.Datum("Unconfigure", unconfigure))
	z.zclCommunicator.UnregisterMatch(z.match)

	if unconfigure {
		if value, ok := z.config.Bool(ReportingConfiguredKey); ok && value {
			ieee, _, ack, seq := z.transmissionLookup(z.device, zigbee.ProfileHomeAutomation)

			if err := z.zclCommunicator.ConfigureReporting(ctx, ieee, ack, z.clusterID, zigbee.NoManufacturer, z.localEndpoint, z.remoteEndpoint, seq, z.attributeID, z.attributeDataType, uint16(0xffff), uint16(0x0000), nil); err != nil {
				z.logger.Error(ctx, "Failed to unconfigure reporting.", logwrap.Err(err))
			}
		}

		z.config.Delete(ReportingConfiguredKey)
		z.config.Delete(PollingConfiguredKey)
	}

	if z.ticker != nil {
		z.pollerStop <- struct{}{}
	}

	return nil
}

func (z *zclMonitor) poller(pctx context.Context) {
	defer close(z.pollerStop)

	for {
		select {
		case <-z.pollerStop:
			z.ticker.Stop()
			z.ticker = nil
			return
		case <-z.ticker.C:
			_, _, ack, seq := z.transmissionLookup(z.device, zigbee.ProfileHomeAutomation)

			ctx, done := context.WithTimeout(pctx, time.Duration(5)*time.Second)
			if _, err := z.zclCommunicator.ReadAttributes(ctx, z.ieeeAddress, ack, z.clusterID, zigbee.NoManufacturer, z.localEndpoint, z.remoteEndpoint, seq, []zcl.AttributeID{z.attributeID}); err != nil {
				z.logger.Error(ctx, "Failed to read attribute.", logwrap.Err(err), logwrap.Datum("ClusterID", z.clusterID), logwrap.Datum("AttributeID", z.attributeID))
			}
			done()
		}
	}
}

func (z *zclMonitor) zclFilter(a zigbee.IEEEAddress, _ zigbee.ApplicationMessage, m zcl.Message) bool {
	return a == z.ieeeAddress &&
		m.SourceEndpoint == z.remoteEndpoint &&
		m.DestinationEndpoint == z.localEndpoint &&
		m.Direction == zcl.ServerToClient
}

func (z *zclMonitor) zclMessage(m communicator.MessageWithSource) {
	switch cmd := m.Message.Command.(type) {
	case *global.ReportAttributes:
		for _, record := range cmd.Records {
			if record.Identifier == z.attributeID && record.DataTypeValue.DataType == z.attributeDataType {
				z.callback(record.Identifier, *record.DataTypeValue)
			}
		}
	case *global.ReadAttributesResponse:
		for _, record := range cmd.Records {
			if record.Identifier == z.attributeID && record.DataTypeValue.DataType == z.attributeDataType && record.Status == 0 {
				z.callback(record.Identifier, *record.DataTypeValue)
			}
		}
	}
}
