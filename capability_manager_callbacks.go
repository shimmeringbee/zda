package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
)

func (m *CapabilityManager) deviceAddedCallback(ctx context.Context, e internalDeviceAdded) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceManagerCapability {
		if err := aC.AddedDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}

func (m *CapabilityManager) deviceRemovedCallback(ctx context.Context, e internalDeviceRemoved) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceManagerCapability {
		if err := aC.RemovedDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}

func (m *CapabilityManager) deviceEnumeratedCallback(pctx context.Context, e internalDeviceEnumeration) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceEnumerationCapability {
		bC, ok := aC.(da.BasicCapability)
		name := "Unknown"

		if ok {
			name = bC.Name()
		}

		ctx, segmentEnd := m.logger.Segment(pctx, "Capability Enumeration", logwrap.Datum("Capability", name))
		if err := aC.EnumerateDevice(ctx, zdaDevice); err != nil {
			m.logger.LogError(ctx, "Enumeration Failed.", logwrap.Err(err))
		}
		segmentEnd()
	}

	return nil
}
