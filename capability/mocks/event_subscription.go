package mocks

import (
	"context"
	. "github.com/shimmeringbee/zda/capability"
	"github.com/stretchr/testify/mock"
)

type MockEventSubscription struct {
	mock.Mock
}

func (m *MockEventSubscription) AddedDevice(f func(context.Context, AddedDevice) error) {
	m.Called(f)
}

func (m *MockEventSubscription) RemovedDevice(f func(context.Context, RemovedDevice) error) {
	m.Called(f)
}

func (m *MockEventSubscription) EnumerateDevice(f func(context.Context, EnumerateDevice) error) {
	m.Called(f)
}
