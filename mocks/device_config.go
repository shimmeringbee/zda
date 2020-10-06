package mocks

import (
	"github.com/shimmeringbee/zda"
	"github.com/stretchr/testify/mock"
	"time"
)

type MockDeviceConfig struct {
	mock.Mock
}

func (m *MockDeviceConfig) Get(d zda.Device, k string) zda.Config {
	args := m.Called(d, k)
	return args.Get(0).(zda.Config)
}

type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) String(k string, d string) string {
	args := m.Called(k, d)
	return args.String(0)
}

func (m *MockConfig) Int(k string, d int) int {
	args := m.Called(k, d)
	return args.Int(0)
}

func (m *MockConfig) Float(k string, d float64) float64 {
	args := m.Called(k, d)
	return args.Get(0).(float64)
}

func (m *MockConfig) Bool(k string, d bool) bool {
	args := m.Called(k, d)
	return args.Bool(0)
}

func (m *MockConfig) Duration(k string, d time.Duration) time.Duration {
	args := m.Called(k, d)
	return args.Get(0).(time.Duration)
}
