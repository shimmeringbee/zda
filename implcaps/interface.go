package implcaps

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zda/attribute"
)

const (
	DataKeyAlreadyInitialised = "Initialised"
	DataKeyZigbeeEndpoint     = "ZigbeeEndpoint"
	DataKeyZigbeeClusterID    = "ZigbeeClusterID"
	DataKeyZCLAttributeID     = "ZCLAttributeID"
)

type DetachType int

const (
	// DeviceRemoved is used when a device has been removed from the Zigbee network, this has already occurred, and it
	// should be assumed that no communication is possible.
	DeviceRemoved DetachType = iota
	// NoLongerEnumerated is used when the enumeration of the node no longer results in this capability existing, or
	// it's being replaced by a different implementation. Tidy up via the network may be possible.
	NoLongerEnumerated
	// FailedAttach is used when an Attach failed.
	FailedAttach
)

type ZDACapability interface {
	// BasicCapability functions should also be present.
	da.BasicCapability
	// Init is used upon creation of the capability to provide persistence.
	Init(da.Device, persistence.Section)
	// Load is used upon load of the capability from persistence at start up.
	Load(context.Context) (bool, error)
	// Enumerate is used to enumerate or re-enumerate a device. Attach should return true if everything is successful
	// and the capability should be attached, or false if it should not. It should also return false if the device has
	// now detached as a result of Enumeration. A return value of true and error is possible, and the capability
	// should attach.
	Enumerate(context.Context, map[string]any) (bool, error)
	// Detach is called when a capability is removed from a device. This will be called after an Attach that returned
	// false, even if it was a new enumeration.
	Detach(context.Context, DetachType) error
	// ImplName returns the implementation name of the capability.
	ImplName() string
}

type ZDAInterface interface {
	// NewAttributeMonitor creates a new attribute monitor to be used to listen to an attribute on a device.
	NewAttributeMonitor() attribute.Monitor
	// SendEvent allows a capability to publish event messages.
	SendEvent(any)
}
