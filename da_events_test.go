package zda

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockEventSender struct {
	mock.Mock
}

func (m *mockEventSender) sendEvent(event interface{}) {
	m.Called(event)
}

func TestZigbeeGateway_ReadEvent(t *testing.T) {
	t.Run("context which expires should result in error", func(t *testing.T) {
		zgw := New(context.Background(), nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := zgw.ReadEvent(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("sent events are received through ReadEvent", func(t *testing.T) {
		zgw := New(context.Background(), nil, nil).(*gateway)

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		expectedEvent := struct{}{}

		go func() {
			zgw.sendEvent(expectedEvent)
		}()

		actualEvent, err := zgw.ReadEvent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvent, actualEvent)
	})
}
