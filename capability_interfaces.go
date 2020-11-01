package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"time"
)

type ProcessingCapability interface {
	Start()
	Stop()
}

type InitableCapability interface {
	Init(CapabilitySupervisor)
}

type PersistableCapability interface {
	da.BasicCapability
	DataStruct() interface{}
	Save(Device) (interface{}, error)
	Load(Device, interface{}) error
}

type FetchCapability interface {
	Get(da.Capability) interface{}
}

type ManageDeviceCapabilities interface {
	Add(Device, da.Capability)
	Remove(Device, da.Capability)
}

type DeviceManagementCapability interface {
	AddedDevice(context.Context, Device) error
	RemovedDevice(context.Context, Device) error
}

type DeviceEnumerationCapability interface {
	EnumerateDevice(context.Context, Device) error
}

type ComposeDADevice interface {
	Compose(Device) da.Device
}

type DeviceLookup interface {
	ByDA(da.Device) (Device, bool)
}

type ZCLFilter func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool
type ZCLCallback func(Device, zcl.Message)

type ZCLCommandLibrary func(*zcl.CommandRegistry)

type ZCL interface {
	RegisterCommandLibrary(ZCLCommandLibrary)
	ReadAttributes(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error)
	WriteAttributes(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, map[zcl.AttributeID]zcl.AttributeDataTypeValue) (map[zcl.AttributeID]global.WriteAttributesResponseRecord, error)
	Bind(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID) error
	ConfigureReporting(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, zcl.AttributeID, zcl.AttributeDataType, uint16, uint16, interface{}) error
	Listen(ZCLFilter, ZCLCallback)
	SendCommand(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, interface{}) error
}

type DAEventSender interface {
	Send(interface{})
}

type Poller interface {
	Add(Device, time.Duration, func(context.Context, Device) bool) func()
}

type Config interface {
	String(string, string) string
	Int(string, int) int
	Float(string, float64) float64
	Bool(string, bool) bool
	Duration(string, time.Duration) time.Duration
}

type DeviceConfig interface {
	Get(Device, string) Config
}

type AttributeMonitor interface {
	Attach(context.Context, Device, zigbee.Endpoint, interface{}) (bool, error)
	Detach(context.Context, Device)
	Reattach(context.Context, Device, zigbee.Endpoint, bool)
	Poll(context.Context, Device)
}

type AttributeMonitorCreator interface {
	Create(da.BasicCapability, zigbee.ClusterID, zcl.AttributeID, zcl.AttributeDataType, func(Device, zcl.AttributeID, zcl.AttributeDataTypeValue)) AttributeMonitor
}

type CapabilitySupervisor interface {
	FetchCapability() FetchCapability
	ManageDeviceCapabilities() ManageDeviceCapabilities
	ComposeDADevice() ComposeDADevice
	DeviceLookup() DeviceLookup
	ZCL() ZCL
	DAEventSender() DAEventSender
	Poller() Poller
	DeviceConfig() DeviceConfig
	AttributeMonitorCreator() AttributeMonitorCreator
}

type SimpleSupervisor struct {
	FCImpl                      FetchCapability
	MDCImpl                     ManageDeviceCapabilities
	CDADImpl                    ComposeDADevice
	DLImpl                      DeviceLookup
	ZCLImpl                     ZCL
	DAESImpl                    DAEventSender
	PollerImpl                  Poller
	DeviceConfigImpl            DeviceConfig
	AttributeMonitorCreatorImpl AttributeMonitorCreator
}

func (s SimpleSupervisor) FetchCapability() FetchCapability {
	return s.FCImpl
}

func (s SimpleSupervisor) ManageDeviceCapabilities() ManageDeviceCapabilities {
	return s.MDCImpl
}

func (s SimpleSupervisor) ComposeDADevice() ComposeDADevice {
	return s.CDADImpl
}

func (s SimpleSupervisor) DeviceLookup() DeviceLookup {
	return s.DLImpl
}

func (s SimpleSupervisor) ZCL() ZCL {
	return s.ZCLImpl
}

func (s SimpleSupervisor) DAEventSender() DAEventSender {
	return s.DAESImpl
}

func (s SimpleSupervisor) Poller() Poller {
	return s.PollerImpl
}

func (s SimpleSupervisor) DeviceConfig() DeviceConfig {
	return s.DeviceConfigImpl
}

func (s SimpleSupervisor) AttributeMonitorCreator() AttributeMonitorCreator {
	return s.AttributeMonitorCreatorImpl
}
