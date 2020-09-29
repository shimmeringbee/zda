package mocks

import (
	"github.com/shimmeringbee/da"
	"github.com/stretchr/testify/mock"
)

type MockFetchCapability struct {
	mock.Mock
}

func (m *MockFetchCapability) Get(c da.Capability) interface{} {
	ret := m.Called(c)
	return ret.Get(0)
}
