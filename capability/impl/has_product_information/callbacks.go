package has_product_information

import (
	"context"
	"github.com/shimmeringbee/zda/capability"
	"github.com/shimmeringbee/zigbee"
)

func (i *Implementation) addedDeviceCallback(ctx context.Context, addedDevice capability.AddedDevice) error {
	ch := make(chan error, 1)
	i.msgCh <- addedDeviceReq{device: addedDevice.Device, ch: ch}

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return zigbee.ContextExpired
	}
}

func (i *Implementation) removedDeviceCallback(ctx context.Context, removedDevice capability.RemovedDevice) error {
	ch := make(chan error, 1)
	i.msgCh <- removedDeviceReq{device: removedDevice.Device, ch: ch}

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return zigbee.ContextExpired
	}
}

func (i *Implementation) enumerateDeviceCallback(ctx context.Context, enumerateDevice capability.EnumerateDevice) error {
	ch := make(chan error, 1)
	i.msgCh <- enumerateDeviceReq{device: enumerateDevice.Device, ch: ch}

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return zigbee.ContextExpired
	}
}
