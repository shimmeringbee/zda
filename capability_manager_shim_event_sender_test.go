package zda

import (
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestCapabilityManager_initSupervisor_DAEventSender(t *testing.T) {
	t.Run("provides an implementation that calls the gateway capabilities for add capability", func(t *testing.T) {
		mes := &mockEventSender{}
		defer mes.AssertExpectations(t)

		mes.On("sendEvent", mock.Anything)

		m := CapabilityManager{eventSender: mes}
		s := m.initSupervisor()

		s.DAEventSender().Send(nil)
	})
}
