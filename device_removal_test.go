package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestZigbeeDeviceRemoval_Remove(t *testing.T) {
	t.Run("successfully calls RequestNodeLeave on provider with devices ieee and Request flag", func(t *testing.T) {
		mockProvider := zigbee.MockProvider{}
		defer mockProvider.AssertExpectations(t)

		expectedIEEE := zigbee.GenerateLocalAdministeredIEEEAddress()

		zed := deviceRemoval{
			nodeRemover: &mockProvider,
			logger:      lw.New(discard.Discard()),
			node:        &node{address: expectedIEEE},
		}

		mockProvider.On("RequestNodeLeave", mock.Anything, expectedIEEE).Return(nil)

		err := zed.Remove(context.Background(), capabilities.Request)
		assert.NoError(t, err)
	})

	t.Run("successfully calls ForceNodeLeave on provider with devices ieee and Force flag", func(t *testing.T) {
		mockProvider := zigbee.MockProvider{}
		defer mockProvider.AssertExpectations(t)

		expectedIEEE := zigbee.GenerateLocalAdministeredIEEEAddress()

		zed := deviceRemoval{
			nodeRemover: &mockProvider,
			logger:      lw.New(discard.Discard()),
			node:        &node{address: expectedIEEE},
		}

		mockProvider.On("ForceNodeLeave", mock.Anything, expectedIEEE).Return(nil)

		err := zed.Remove(context.Background(), capabilities.Force)
		assert.NoError(t, err)
	})
}
