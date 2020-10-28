package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestCapabilityManager_deviceAddedCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("Name").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("AddedDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceAddedCallback(ctx, internalDeviceAdded{device: device})
		assert.NoError(t, err)

	})
}

func TestCapabilityManager_deviceRemoveCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("Name").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("RemovedDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceRemovedCallback(ctx, internalDeviceRemoved{device: device})
		assert.NoError(t, err)

	})
}

func TestCapabilityManager_deviceEnumeratedCallback(t *testing.T) {
	t.Run("calls any capabilities that have the device management interface", func(t *testing.T) {
		m := CapabilityManager{
			capabilityByFlag:    map[da.Capability]interface{}{},
			capabilityByKeyName: map[string]PersistableCapability{},
		}

		f := da.Capability(0x0000)
		kN := "KEY"

		mC := &mockCapability{}
		mC.On("Capability").Return(f)
		mC.On("Name").Return(kN)

		m.Add(mC)

		node := &internalNode{
			mutex:                &sync.RWMutex{},
			endpointDescriptions: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}
		device := &internalDevice{
			node:  node,
			mutex: &sync.RWMutex{},
		}
		zdaDevice := Device{
			Endpoints: map[zigbee.Endpoint]zigbee.EndpointDescription{},
		}

		ctx := context.TODO()

		mC.On("EnumerateDevice", ctx, zdaDevice).Return(nil)
		defer mC.AssertExpectations(t)

		err := m.deviceEnumeratedCallback(ctx, internalDeviceEnumeration{device: device})
		assert.NoError(t, err)

	})
}
