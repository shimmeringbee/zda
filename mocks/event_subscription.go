package mocks

import (
	"context"
	"github.com/shimmeringbee/zda"
	"github.com/stretchr/testify/mock"
)

type MockEventSubscription struct {
	mock.Mock
}

func (m *MockEventSubscription) AddedDevice(f func(context.Context, zda.AddedDeviceEvent) error) {
	m.Called(f)
}

func (m *MockEventSubscription) RemovedDevice(f func(context.Context, zda.RemovedDeviceEvent) error) {
	m.Called(f)
}

func (m *MockEventSubscription) EnumerateDevice(f func(context.Context, zda.EnumerateDeviceEvent) error) {
	m.Called(f)
}
