package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testGatewayIEEEAddress = zigbee.GenerateLocalAdministeredIEEEAddress()

func newTestGateway() (*gateway, *zigbee.MockProvider, func(*testing.T)) {
	mp := new(zigbee.MockProvider)

	mp.On("AdapterNode").Return(zigbee.Node{
		IEEEAddress: testGatewayIEEEAddress,
	}).Maybe()

	gw := New(context.Background(), mp)

	return gw.(*gateway), mp, func(t *testing.T) {
		err := gw.Stop(nil)
		assert.NoError(t, err)
		mp.AssertExpectations(t)
	}
}

func Test_gateway_New(t *testing.T) {
	t.Run("calling the new constructor returns a valid gateway, with the zigbee.Provider specified", func(t *testing.T) {
		gw, mp, stop := newTestGateway()
		defer stop(t)

		assert.NotNil(t, gw)
		assert.Equal(t, mp, gw.provider)
	})
}

func Test_gateway_Start(t *testing.T) {
	t.Run("Initialises state from the zigbee.Provider and returns a Self device with valid information", func(t *testing.T) {
		gw, _, stop := newTestGateway()
		defer stop(t)

		err := gw.Start(nil)
		assert.NoError(t, err)

		self := gw.Self()
		assert.Equal(t, testGatewayIEEEAddress, self.Identifier())
		assert.Equal(t, gw, self.Gateway())
		assert.Contains(t, self.Capabilities(), capabilities.DeviceDiscoveryFlag)
	})
}
