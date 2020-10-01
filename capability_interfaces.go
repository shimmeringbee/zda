package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"time"
)

type BasicCapability interface {
	Capability() da.Capability
}

type ProcessingCapability interface {
	Start()
	Stop()
}

type InitableCapability interface {
	Init(CapabilitySupervisor)
}

type PersistableCapability interface {
	BasicCapability
	KeyName() string
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

type CapabilitySupervisor interface {
	FetchCapability() FetchCapability
	ManageDeviceCapabilities() ManageDeviceCapabilities
	ComposeDADevice() ComposeDADevice
	DeviceLookup() DeviceLookup
	ZCL() ZCL
	DAEventSender() DAEventSender
	Poller() Poller
}

type SimpleSupervisor struct {
	FCImpl     FetchCapability
	MDCImpl    ManageDeviceCapabilities
	CDADImpl   ComposeDADevice
	DLImpl     DeviceLookup
	ZCLImpl    ZCL
	DAESImpl   DAEventSender
	PollerImpl Poller
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
