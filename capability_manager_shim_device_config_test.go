package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func Test_deviceConfigShim(t *testing.T) {
	t.Run("passes through string query to rule", func(t *testing.T) {
		ns := "ns"
		key := "key"

		expected := "string"

		rBC := ruleBaseConfig{
			r: rules.Rule{
				Settings: map[string]rules.Settings{
					ns: {
						key: expected,
					},
				},
			},
			ns: ns,
		}

		actual := rBC.String(key, "def")
		assert.Equal(t, expected, actual)
	})

	t.Run("passes through int query to rule", func(t *testing.T) {
		ns := "ns"
		key := "key"

		expected := 5

		rBC := ruleBaseConfig{
			r: rules.Rule{
				Settings: map[string]rules.Settings{
					ns: {
						key: expected,
					},
				},
			},
			ns: ns,
		}

		actual := rBC.Int(key, 10)
		assert.Equal(t, expected, actual)
	})

	t.Run("passes through float query to rule", func(t *testing.T) {
		ns := "ns"
		key := "key"

		expected := 1.0

		rBC := ruleBaseConfig{
			r: rules.Rule{
				Settings: map[string]rules.Settings{
					ns: {
						key: expected,
					},
				},
			},
			ns: ns,
		}

		actual := rBC.Float(key, 2.0)
		assert.Equal(t, expected, actual)
	})

	t.Run("passes through boolean query to rule", func(t *testing.T) {
		ns := "ns"
		key := "key"

		expected := true

		rBC := ruleBaseConfig{
			r: rules.Rule{
				Settings: map[string]rules.Settings{
					ns: {
						key: expected,
					},
				},
			},
			ns: ns,
		}

		actual := rBC.Bool(key, false)
		assert.Equal(t, expected, actual)
	})

	t.Run("passes through duration query to rule", func(t *testing.T) {
		ns := "ns"
		key := "key"

		expected := 1 * time.Second

		rBC := ruleBaseConfig{
			r: rules.Rule{
				Settings: map[string]rules.Settings{
					ns: {
						key: 1000,
					},
				},
			},
			ns: ns,
		}

		actual := rBC.Duration(key, 2*time.Second)
		assert.Equal(t, expected, actual)
	})
}

type MockFetchCapability struct {
	mock.Mock
}

func (m *MockFetchCapability) Get(c da.Capability) interface{} {
	ret := m.Called(c)
	return ret.Get(0)
}

type MockHasProductInformation struct {
	mock.Mock
}

func (m *MockHasProductInformation) ProductInformation(ctx context.Context, d da.Device) (capabilities.ProductInformation, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(capabilities.ProductInformation), args.Error(1)
}

func Test_deviceConfigShim_Get(t *testing.T) {
	t.Run("returns a matching rule based config with the correct ns", func(t *testing.T) {
		mFC := &MockFetchCapability{}
		defer mFC.AssertExpectations(t)

		cDA := &ComposeDADeviceShim{}

		mFC.On("Get", capabilities.HasProductInformationFlag).Return(nil)

		nt, _, iDevs := generateNodeTableWithData(1)
		iDev := iDevs[0]

		dad := internalDeviceToZDADevice(iDev)

		ns := "ns"
		key := "key"
		expected := "val"

		s := deviceConfigShim{
			ruleList: &rules.Rule{
				Filter: rules.Filter{},
				Settings: map[string]rules.Settings{
					ns: {
						key: expected,
					},
				},
			},
			capabilityFetcher: mFC,
			composeDADevice:   cDA,
			nodeTable:         nt,
		}

		cfg := s.Get(dad, ns)

		actual := cfg.String(key, "def")

		assert.Equal(t, expected, actual)
	})
}

func Test_deviceConfigShim_constructMatchData(t *testing.T) {
	t.Run("constructs a MatchData from the device, calling the HasProductInformation capability", func(t *testing.T) {
		mFC := &MockFetchCapability{}
		defer mFC.AssertExpectations(t)

		mHPI := &MockHasProductInformation{}
		defer mHPI.AssertExpectations(t)

		cDA := &ComposeDADeviceShim{}

		nt, iNode, iDevs := generateNodeTableWithData(2)
		iDev := iDevs[1]

		d := internalDeviceToZDADevice(iDev)

		dad := cDA.Compose(d)

		pi := capabilities.ProductInformation{
			Manufacturer: "manu",
			Name:         "product",
		}

		mFC.On("Get", capabilities.HasProductInformationFlag).Return(mHPI)
		mHPI.On("ProductInformation", mock.Anything, dad).Return(pi, nil)

		s := deviceConfigShim{
			ruleList:          nil,
			capabilityFetcher: mFC,
			composeDADevice:   cDA,
			nodeTable:         nt,
		}

		expected := rules.MatchData{
			ManufacturerCode: iNode.nodeDesc.ManufacturerCode,
			ManufacturerName: pi.Manufacturer,
			ProductName:      pi.Name,
			DeviceId:         iDev.deviceID,
		}

		actual := s.constructMatchData(d)

		assert.Equal(t, expected, actual)
	})
}
