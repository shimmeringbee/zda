package zda

import (
	"context"
	"errors"
	"github.com/shimmeringbee/da"
	. "github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestZigbeeDeviceDiscovery_Contract(t *testing.T) {
	t.Run("can be assigned to a capability.DeviceDiscovery", func(t *testing.T) {
		assert.Implements(t, (*DeviceDiscovery)(nil), new(ZigbeeDeviceDiscovery))
	})
}

func TestZigbeeDeviceDiscovery_Enable(t *testing.T) {
	t.Run("calling enable on Device which is not the gateway self errors", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice)

		zdd := ZigbeeDeviceDiscovery{gateway: &mockGateway}

		nonSelfDevice := da.Device{}

		err := zdd.Enable(context.Background(), nonSelfDevice, 500*time.Millisecond)
		assert.Error(t, err)

		mockGateway.AssertExpectations(t)
	})

	t.Run("calling enable on device which is self causes AllowJoin of zigbee provider", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.IsType(DeviceDiscoveryEnabled{}))

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}
		defer zdd.Stop()

		err := zdd.Enable(context.Background(), gatewayDevice, 500*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background(), gatewayDevice)
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("calling enable on device which is self causes AllowJoin of zigbee provider, and forwards an error", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}

		expectedError := errors.New("error")
		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(expectedError)

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}
		defer zdd.Stop()

		err := zdd.Enable(context.Background(), zdd.gateway.Self(), 500*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)

		status, err := zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}

func TestZigbeeDeviceDiscovery_Disable(t *testing.T) {
	t.Run("calling disable on device which is not the gateway self errors", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice)

		zdd := ZigbeeDeviceDiscovery{gateway: &mockGateway}

		nonSelfDevice := da.Device{}

		err := zdd.Disable(context.Background(), nonSelfDevice)
		assert.Error(t, err)

		mockGateway.AssertExpectations(t)
	})

	t.Run("calling disable on device which is self causes DenyJoin of zigbee provider", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.IsType(DeviceDiscoveryDisabled{}))

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil)

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}
		defer zdd.Stop()

		zdd.discovering = true

		err := zdd.Disable(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("calling disable on device which is self causes DenyJoin of zigbee provider, and forwards an error", func(t *testing.T) {
		expectedError := errors.New("deny join failure")

		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(expectedError)

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}
		defer zdd.Stop()

		zdd.discovering = true

		err := zdd.Disable(context.Background(), zdd.gateway.Self())

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)

		status, err := zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}

func TestZigbeeDeviceDiscovery_DurationBehaviour(t *testing.T) {
	t.Run("when an allows duration expires then a disable instruction is sent", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything).Return(nil).Twice()

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil)

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}

		defer zdd.Stop()

		err := zdd.Enable(context.Background(), zdd.gateway.Self(), 100*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		time.Sleep(150 * time.Millisecond)

		status, err = zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.False(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})

	t.Run("second allows extend the duration of the first", func(t *testing.T) {
		mockGateway := mockGateway{}
		gatewayDevice := da.Device{
			Gateway:    &mockGateway,
			Identifier: zigbee.IEEEAddress(0x01),
		}
		mockGateway.On("Self").Return(gatewayDevice).Maybe()

		mockEventSender := mockEventSender{}
		mockEventSender.On("sendEvent", mock.Anything).Return(nil).Twice()

		mockNetworkJoining := mockNetworkJoining{}
		mockNetworkJoining.On("PermitJoin", mock.Anything, true).Return(nil)
		mockNetworkJoining.On("DenyJoin", mock.Anything).Return(nil).Maybe()

		zdd := ZigbeeDeviceDiscovery{
			gateway:        &mockGateway,
			eventSender:    &mockEventSender,
			networkJoining: &mockNetworkJoining,
		}

		defer zdd.Stop()

		err := zdd.Enable(context.Background(), zdd.gateway.Self(), 50*time.Millisecond)
		assert.NoError(t, err)

		status, err := zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)
		assert.Greater(t, int64(status.RemainingDuration), int64(45*time.Millisecond))

		err = zdd.Enable(context.Background(), zdd.gateway.Self(), 200*time.Millisecond)
		assert.NoError(t, err)

		status, err = zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)
		assert.Greater(t, int64(status.RemainingDuration), int64(145*time.Millisecond))

		time.Sleep(150 * time.Millisecond)

		status, err = zdd.Status(context.Background(), zdd.gateway.Self())
		assert.NoError(t, err)
		assert.True(t, status.Discovering)

		mockGateway.AssertExpectations(t)
		mockEventSender.AssertExpectations(t)
		mockNetworkJoining.AssertExpectations(t)
	})
}
