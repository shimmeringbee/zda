package capability

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
)

type FetchCapability interface {
	Get(da.Capability) interface{}
}

type ManageDeviceCapabilities interface {
	Add(Device, da.Capability)
	Remove(Device, da.Capability)
}

type EventSubscription interface {
	AddedDevice(func(context.Context, AddedDevice) error)
	RemovedDevice(func(context.Context, RemovedDevice) error)
	EnumerateDevice(func(context.Context, EnumerateDevice) error)
}

type ComposeDADevice interface {
	Compose(Device) da.Device
}

type DeviceLookup interface {
	ByDA(da.Device) (Device, bool)
}

type ZCL interface {
	ReadAttributes(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, []zcl.AttributeID) (map[zcl.AttributeID]global.ReadAttributeResponseRecord, error)
	//Bind(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID) error
	//ConfigureReporting(context.Context, Device, zigbee.Endpoint, zigbee.ClusterID, zcl.AttributeID, zcl.AttributeDataType, uint16, uint16, interface{}) error
}

type Supervisor interface {
	FetchCapability() FetchCapability
	ManageDeviceCapabilities() ManageDeviceCapabilities
	EventSubscription() EventSubscription
	ComposeDADevice() ComposeDADevice
	DeviceLookup() DeviceLookup
	ZCL() ZCL
}

type SimpleSupervisor struct {
	FCImpl   FetchCapability
	MDCImpl  ManageDeviceCapabilities
	ESImpl   EventSubscription
	CDADImpl ComposeDADevice
	DLImpl   DeviceLookup
	ZCLImpl  ZCL
}

func (s SimpleSupervisor) FetchCapability() FetchCapability {
	return s.FCImpl
}

func (s SimpleSupervisor) ManageDeviceCapabilities() ManageDeviceCapabilities {
	return s.MDCImpl
}

func (s SimpleSupervisor) EventSubscription() EventSubscription {
	return s.ESImpl
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

type BasicCapability interface {
	Capability() da.Capability
}

type ProcessingCapability interface {
	Start()
	Stop()
}

type InitableCapability interface {
	Init(Supervisor)
}

type PersistableCapability interface {
	BasicCapability
	KeyName() string
	DataStruct() interface{}
	Save(Device) (interface{}, error)
	Load(Device, interface{}) error
}
