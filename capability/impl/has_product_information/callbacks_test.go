package has_product_information

import (
	"context"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zda/capability/mocks"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestImplementation_handleAddedDevice(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)

		i.Init(mockSupervisor)
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.handleAddedDevice(ctx, capability.AddedDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("updates internal store to have a device with empty data", func(t *testing.T) {
		i := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)

		i.Init(mockSupervisor)
		i.Start()
		defer i.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		id := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		err := i.handleAddedDevice(ctx, capability.AddedDevice{Device: capability.Device{
			Identifier:   id,
			Capabilities: nil,
			Endpoints:    nil,
		}})
		assert.NoError(t, err)

		internalData, found := i.data[id]
		assert.True(t, found)
		assert.Nil(t, internalData.Manufacturer)
		assert.Nil(t, internalData.Product)
	})

}

func TestImplementation_handleRemovedDevice(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)

		i.Init(mockSupervisor)
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.handleRemovedDevice(ctx, capability.RemovedDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("updates internal store to have a device with empty data", func(t *testing.T) {
		i := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)

		i.Init(mockSupervisor)
		i.Start()
		defer i.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		id := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		err := i.handleAddedDevice(ctx, capability.AddedDevice{Device: capability.Device{
			Identifier:   id,
			Capabilities: nil,
			Endpoints:    nil,
		}})
		assert.NoError(t, err)

		err = i.handleRemovedDevice(ctx, capability.RemovedDevice{Device: capability.Device{
			Identifier:   id,
			Capabilities: nil,
			Endpoints:    nil,
		}})
		assert.NoError(t, err)

		_, found := i.data[id]
		assert.False(t, found)
	})
}

func TestImplementation_handleEnumerateDevice(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}

		mockSupervisor := &mocks.MockSupervisor{}
		mockEventSubscription := &mocks.MockEventSubscription{}

		mockSupervisor.On("EventSubscription").Return(mockEventSubscription)

		mockEventSubscription.On("AddedDevice", mock.Anything)
		mockEventSubscription.On("RemovedDevice", mock.Anything)
		mockEventSubscription.On("EnumerateDevice", mock.Anything)

		i.Init(mockSupervisor)
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.handleEnumerateDevice(ctx, capability.EnumerateDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

}
