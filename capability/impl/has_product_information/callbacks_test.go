package has_product_information

import (
	"context"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestImplementation_addedDeviceCallback(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.addedDeviceCallback(ctx, capability.AddedDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("returns when reply channel receives a value", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		id := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		expectedDevice := capability.Device{
			Identifier: id,
		}

		go func() {
			msg := (<-i.msgCh).(addedDeviceReq)
			assert.Equal(t, expectedDevice, msg.device)
			msg.ch <- nil
		}()

		err := i.addedDeviceCallback(ctx, capability.AddedDevice{
			Device: expectedDevice,
		})
		assert.NoError(t, err)
	})

}

func TestImplementation_removedDeviceCallback(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.removedDeviceCallback(ctx, capability.RemovedDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("returns when reply channel receives a value", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		id := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		expectedDevice := capability.Device{
			Identifier: id,
		}

		go func() {
			msg := (<-i.msgCh).(removedDeviceReq)
			assert.Equal(t, expectedDevice, msg.device)
			msg.ch <- nil
		}()

		err := i.removedDeviceCallback(ctx, capability.RemovedDevice{
			Device: expectedDevice,
		})
		assert.NoError(t, err)
	})
}

func TestImplementation_enumerateDeviceCallback(t *testing.T) {
	t.Run("unresponded results in context expiry error", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := i.enumerateDeviceCallback(ctx, capability.EnumerateDevice{})
		assert.Equal(t, zigbee.ContextExpired, err)
	})

	t.Run("returns when reply channel receives a value", func(t *testing.T) {
		i := &Implementation{}
		i.msgCh = make(chan interface{}, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		id := zda.IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x00}

		expectedDevice := capability.Device{
			Identifier: id,
		}

		go func() {
			msg := (<-i.msgCh).(enumerateDeviceReq)
			assert.Equal(t, expectedDevice, msg.device)
			msg.ch <- nil
		}()

		err := i.enumerateDeviceCallback(ctx, capability.EnumerateDevice{
			Device: expectedDevice,
		})
		assert.NoError(t, err)
	})
}
