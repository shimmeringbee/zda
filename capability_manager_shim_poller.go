package zda

import (
	"context"
	"time"
)

type pollerShim struct {
	poller poller
}

func (s *pollerShim) Add(d Device, t time.Duration, f func(context.Context, Device) bool) func() {
	isCancelled := false

	s.poller.Add(d.Identifier, t, func(ctx context.Context, iDev *internalDevice) bool {
		if !isCancelled {
			return f(ctx, internalDeviceToZDADevice(iDev))
		} else {
			return false
		}
	})

	return func() {
		isCancelled = true
	}
}
