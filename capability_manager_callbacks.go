package zda

import "context"

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

func (m *CapabilityManager) deviceEnumeratedCallback(ctx context.Context, e internalDeviceEnumeration) error {
	zdaDevice := internalDeviceToZDADevice(e.device)

	for _, aC := range m.deviceEnumerationCapability {
		if err := aC.EnumerateDevice(ctx, zdaDevice); err != nil {
			return err
		}
	}

	return nil
}
