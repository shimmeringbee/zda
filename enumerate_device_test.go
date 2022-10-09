package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/basic"
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

type mockReadAttribute struct {
	mock.Mock
}

func (m *mockReadAttribute) ReadAttribute(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributes)
	return args.Get(0).([]global.ReadAttributeResponseRecord), args.Error(1)
}

func Test_enumerateDevice_interrogateNode(t *testing.T) {
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
				InClusterList:  []zigbee.ClusterID{zcl.BasicId},
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

		mra := &mockReadAttribute{}
		defer mra.AssertExpectations(t)
		mra.On("ReadAttribute", mock.Anything, expectedAddr, false, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), uint8(0), []zcl.AttributeID{basic.ManufacturerName, basic.ModelIdentifier, basic.ManufacturerVersionDetails, basic.SerialNumber}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: basic.ManufacturerName,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "manufacturer",
					},
				},
				{
					Identifier: basic.ModelIdentifier,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "model",
					},
				},
				{
					Identifier: basic.ManufacturerVersionDetails,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "version",
					},
				},
				{
					Identifier: basic.SerialNumber,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "serial",
					},
				},
			}, nil)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq, zclReadFn: mra.ReadAttribute}
		n := &node{address: expectedAddr, sequence: makeTransactionSequence()}

		inv, err := ed.interrogateNode(context.Background(), n)
		assert.NoError(t, err)

		assert.Equal(t, expectedNodeDescription, *inv.description)
		assert.Equal(t, expectedEndpointDescs[0], inv.endpoints[0x01].description)
		assert.Equal(t, expectedEndpointDescs[1], inv.endpoints[0x02].description)

		assert.Equal(t, "model", inv.endpoints[0x01].productInformation.product)
		assert.Equal(t, "serial", inv.endpoints[0x01].productInformation.serial)
		assert.Equal(t, "version", inv.endpoints[0x01].productInformation.version)
		assert.Equal(t, "manufacturer", inv.endpoints[0x01].productInformation.manufacturer)
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
