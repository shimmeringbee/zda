package mocks

import (
	"context"
	"github.com/shimmeringbee/zda"
	"github.com/stretchr/testify/mock"
	"time"
)

type MockPoller struct {
	mock.Mock
}

func (m *MockPoller) Add(d zda.Device, t time.Duration, f func(context.Context, zda.Device) bool) func() {
	args := m.Called(d, t, f)
	return args.Get(0).(func())
}
