package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockNetworkJoining struct {
	mock.Mock
}

func (m *mockNetworkJoining) PermitJoin(ctx context.Context, allRouters bool) error {
	args := m.Called(ctx, allRouters)
	return args.Error(0)
}

func (m *mockNetworkJoining) DenyJoin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestZigbeeDeviceDiscovery_Enable(t *testing.T) {
	t.Run("calling enable on device which is self causes AllowJoin of zigbee provider", func(t *testing.T) {
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.IsType(capabilities.DeviceDiscoveryEnabled{}))

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}
		defer zdd.Stop()

		err := zdd.Enable(context.Background(), 500*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("calling enable on device which is self causes AllowJoin of zigbee provider, and forwards an error", func(t *testing.T) {
		mockEventSender := mockEventSender{}

		expectedError := errors.New("error")
		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(expectedError)

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}
		defer zdd.Stop()

		err := zdd.Enable(context.Background(), 500*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}

func TestZigbeeDeviceDiscovery_Disable(t *testing.T) {
	t.Run("calling disable on device which is self causes DenyJoin of zigbee provider", func(t *testing.T) {
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.IsType(capabilities.DeviceDiscoveryDisabled{}))

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil)

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}
		defer zdd.Stop()

		zdd.discovering = true

		err := zdd.Disable(context.Background())
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("calling disable on device which is self causes DenyJoin of zigbee provider, and forwards an error", func(t *testing.T) {
		expectedError := errors.New("deny join failure")

		mockEventSender := mockEventSender{}

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(expectedError)

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}
		defer zdd.Stop()

		zdd.discovering = true

		err := zdd.Disable(context.Background())

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}

func TestZigbeeDeviceDiscovery_DurationBehaviour(t *testing.T) {
	t.Run("when an allows duration expires then a disable instruction is sent", func(t *testing.T) {
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything).Return(nil).Twice()

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil)

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}

		defer zdd.Stop()

		err := zdd.Enable(context.Background(), 100*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		time.Sleep(150 * time.Millisecond)

		status, err = zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("second allows extend the duration of the first", func(t *testing.T) {
		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything).Return(nil).Twice()

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil).Maybe()

		zdd := deviceDiscovery{
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
			logger:         logwrap.New(discard.Discard()),
		}

		defer zdd.Stop()

		err := zdd.Enable(context.Background(), 50*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)
		assert.Greater(t, int64(status.RemainingDuration), int64(45*time.Millisecond))

		err = zdd.Enable(context.Background(), 200*time.Millisecond)
		assert.NoError(t, err)

		status, err = zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)
		assert.Greater(t, int64(status.RemainingDuration), int64(145*time.Millisecond))

		time.Sleep(150 * time.Millisecond)

		status, err = zdd.Status(context.Background())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}
