package catabilities

import (
	"context"
	"github.com/shimmeringbee/da"
)

const (
	DataKeyAlreadyInitialised = "Initialised"
	DataKeyZigbeeEndpoint     = "ZigbeeEndpoint"
	DataKeyZigbeeClusterID    = "ZigbeeClusterID"
	DataKeyZCLAttributeID     = "ZCLAttributeID"
)

type AttachType int

const (
	// Enumeration is used for Attach when the capability is being created or updated through enumerateDevice.
	Enumeration AttachType = iota
	// Load is used to indicate that state is being loaded from disk. Any Zigbee network configuration should
	// be assumed to be complete.
	Load
)

type ZDACapability interface {
	// Attach is used to initial create, re-enumerate or load a capability on a device. The AttachType guides
	// the capability in determining what to do. Attach should return true if everything is successful and the
	// capability should be attached, or false if it should not. It should also return false if the device has
	// now detached as a result of Enumeration. A return value of true and error is possible, and the capability
	// should attach.
	Attach(context.Context, da.Device, AttachType, map[string]interface{}) (bool, error)
	// State returns a data structure that should be passed to Attach with AttachType.LOAD to reload the capability
	// from a persistent store.
	State() map[string]interface{}
}
