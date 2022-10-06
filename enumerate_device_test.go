package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"
	"io"
	"sync"
	"testing"
)

func Test_enumerateDevice_startEnumeration(t *testing.T) {
	t.Run("returns an error if the node is already being enumerated", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard())}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		n.enumerationSem.TryAcquire(1)
		err := ed.startEnumeration(context.Background(), n)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "enumeration already in progress")
	})

	t.Run("returns nil if node is not being enumerated, and marks the node in progress", func(t *testing.T) {
		mnq := &mockNodeQuerier{}
		defer mnq.AssertExpectations(t)
		mnq.On("QueryNodeDescription", mock.Anything, mock.Anything).Return(zigbee.NodeDescription{}, io.ErrUnexpectedEOF).Maybe()

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		err := ed.startEnumeration(context.Background(), n)
		assert.Nil(t, err)
		assert.False(t, n.enumerationSem.TryAcquire(1))
	})
}

func Test_enumerateDevice_enumerate(t *testing.T) {
	t.Run("interrogates a node for description, endpoints and subsequent endpoint descriptions", func(t *testing.T) {
		expectedAddr := zigbee.GenerateLocalAdministeredIEEEAddress()

		expectedNodeDescription := zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		}
		expectedEndpoints := []zigbee.Endpoint{0x01, 0x02}

		expectedEndpointDescs := []zigbee.EndpointDescription{
			{
				Endpoint:       0x01,
				ProfileID:      0x02,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  nil,
				OutClusterList: nil,
			},
			{
				Endpoint:       0x02,
				ProfileID:      0x03,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  nil,
				OutClusterList: nil,
			},
		}

		mnq := &mockNodeQuerier{}
		defer mnq.AssertExpectations(t)
		mnq.On("QueryNodeDescription", mock.Anything, expectedAddr).Return(expectedNodeDescription, nil)
		mnq.On("QueryNodeEndpoints", mock.Anything, expectedAddr).Return(expectedEndpoints, nil)
		mnq.On("QueryNodeEndpointDescription", mock.Anything, expectedAddr, zigbee.Endpoint(0x01)).Return(expectedEndpointDescs[0], nil)
		mnq.On("QueryNodeEndpointDescription", mock.Anything, expectedAddr, zigbee.Endpoint(0x02)).Return(expectedEndpointDescs[1], nil)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq}

		n := &node{address: expectedAddr}

		inv, err := ed.interrogateNode(context.Background(), n)
		assert.NoError(t, err)

		assert.Equal(t, expectedNodeDescription, *inv.desc)
		assert.Equal(t, expectedEndpoints, inv.endpoints)
		assert.Equal(t, expectedEndpointDescs[0], inv.endpointDesc[0x01].endpointDescription)
		assert.Equal(t, expectedEndpointDescs[1], inv.endpointDesc[0x02].endpointDescription)
	})
}

type mockNodeQuerier struct {
	mock.Mock
}

func (m *mockNodeQuerier) QueryNodeDescription(ctx context.Context, networkAddress zigbee.IEEEAddress) (zigbee.NodeDescription, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).(zigbee.NodeDescription), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpoints(ctx context.Context, networkAddress zigbee.IEEEAddress) ([]zigbee.Endpoint, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).([]zigbee.Endpoint), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpointDescription(ctx context.Context, networkAddress zigbee.IEEEAddress, endpoint zigbee.Endpoint) (zigbee.EndpointDescription, error) {
	args := m.Called(ctx, networkAddress, endpoint)
	return args.Get(0).(zigbee.EndpointDescription), args.Error(1)
}
