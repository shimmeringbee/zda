package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/persistence/impl/memory"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"testing"
)

var testGatewayIEEEAddress = zigbee.GenerateLocalAdministeredIEEEAddress()

func newTestGateway() (*gateway, *zigbee.MockProvider, *mock.Call, func(*testing.T)) {
	mp := new(zigbee.MockProvider)

	mp.On("AdapterNode").Return(zigbee.Node{
		IEEEAddress: testGatewayIEEEAddress,
	}).Maybe()

	mRE := mp.On("ReadEvent", mock.Anything).Return(nil, context.Canceled).Maybe()

	gw := New(context.Background(), memory.New(), mp, nil)

	gw.(*gateway).WithLogWrapLogger(logwrap.New(discard.Discard()))

	return gw.(*gateway), mp, mRE, func(t *testing.T) {
		err := gw.Stop(nil)
		assert.NoError(t, err)
		mp.AssertExpectations(t)
	}
}

func Test_gateway_New(t *testing.T) {
	t.Run("calling the new constructor returns a valid gateway, with the zigbee.Provider specified", func(t *testing.T) {
		gw, mp, _, stop := newTestGateway()
		defer stop(t)

		assert.NotNil(t, gw)
		assert.Equal(t, mp, gw.provider)
	})
}

func Test_gateway_Start(t *testing.T) {
	t.Run("initialises state from the zigbee.Provider, registers endpoints and returns a Self device with valid information", func(t *testing.T) {
		gw, mp, _, stop := newTestGateway()
		defer stop(t)

		mp.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := gw.Start(nil)
		assert.NoError(t, err)

		self := gw.Self()
		assert.Equal(t, testGatewayIEEEAddress, self.Identifier())
		assert.Equal(t, gw, self.Gateway())
		assert.Contains(t, self.Capabilities(), capabilities.DeviceDiscoveryFlag)
	})
}

func Test_gateway_Stop(t *testing.T) {
	t.Run("cancels the context upon call", func(t *testing.T) {
		gw, mp, _, _ := newTestGateway()

		mp.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := gw.Start(nil)
		assert.NoError(t, err)

		assert.NoError(t, gw.ctx.Err())

		err = gw.Stop(nil)
		assert.NoError(t, err)

		assert.ErrorIs(t, gw.ctx.Err(), context.Canceled)
	})
}

func Test_gateway_Devices(t *testing.T) {
	t.Run("returns any devices plus gateway self", func(t *testing.T) {
		gw, mp, _, stop := newTestGateway()
		defer stop(t)

		mp.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mp.On("QueryNodeDescription", mock.Anything, mock.Anything).Return(zigbee.NodeDescription{}, io.EOF).Maybe()

		err := gw.Start(nil)
		assert.NoError(t, err)

		addr := zigbee.GenerateLocalAdministeredIEEEAddress()
		gw.receiveNodeJoinEvent(zigbee.NodeJoinEvent{Node: zigbee.Node{IEEEAddress: addr}})

		devices := gw.Devices()
		assert.Len(t, devices, 2)
		assert.Contains(t, devices, gw.Self())
	})
}
