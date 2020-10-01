package zda

import (
	"context"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestCapabilityManager_initSupervisor_Poller(t *testing.T) {
	t.Run("calls add on the parent poller with identifier", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		cFn := s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			return true
		})

		assert.NotNil(t, cFn)
	})

	t.Run("when capability provided function is called a populated zda is provided", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		called := false

		_, _, iDev := generateNodeTableWithData(1)

		s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			assert.Equal(t, iDev[0].generateIdentifier(), device.Identifier)
			called = true
			return true
		})

		wrappedFn, ok := mockPoller.Calls[0].Arguments[2].(func(context.Context, *internalDevice) bool)
		assert.True(t, ok)

		ret := wrappedFn(context.TODO(), iDev[0])

		assert.True(t, called)
		assert.True(t, ret)
	})

	t.Run("when poller is cancelled the wrapper returns false without calling the wrapped function", func(t *testing.T) {
		mockPoller := &mockPoller{}
		defer mockPoller.AssertExpectations(t)

		m := CapabilityManager{poller: mockPoller}
		s := m.initSupervisor()

		addr := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0x11}
		zdaDevice := Device{Identifier: addr}

		interval := 10 * time.Millisecond

		mockPoller.On("Add", addr, interval, mock.Anything)

		_, _, iDev := generateNodeTableWithData(1)

		cFn := s.Poller().Add(zdaDevice, interval, func(ctx context.Context, device Device) bool {
			t.Fatalf("should not have run wrapper")
			return true
		})

		wrappedFn, ok := mockPoller.Calls[0].Arguments[2].(func(context.Context, *internalDevice) bool)
		assert.True(t, ok)

		cFn()
		ret := wrappedFn(context.TODO(), iDev[0])

		assert.False(t, ret)
	})
}
